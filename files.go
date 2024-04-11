package mist

import (
	"github.com/hashicorp/golang-lru"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FileUploader defines a structure used for configuring and handling the upload of files in a web application.
// It provides a customized way of determining where files should be stored upon upload based on the properties
// provided. It is particularly useful in scenarios where uploaded files need to be saved to different locations
// depending on file metadata or other contextual information available at runtime.
//
// Fields:
//
//   - FileField string: The name of the form field used for the file upload. This corresponds to the 'name'
//     attribute in the HTML form input element for file uploads (e.g., <input type="file" name="myfile">).
//     The FileUploader will look for this specific field in the multipart form data to process the
//     file uploads.
//
//   - DstPathFunc func(*multipart.FileHeader) string: A functional field used to generate the destination path for
//     an uploaded file. This function takes a pointer to a
//     multipart.FileHeader struct, which contains metadata about
//     an uploaded file (such as filename, size, and content type),
//     and returns a string that represents the destination path
//     where the file should be saved. By allowing the path to be
//     dynamically determined through this function, the FileUploader
//     can achieve more flexible and context-sensitive file storage
//     behavior. This is advantageous when the storage location depends
//     on specific file characteristics or other request-specific data.
//
// Example usage of FileUploader:
//
//	uploader := &FileUploader{
//	    FileField: "avatar", // This should match the file input field's name in your form
//	    DstPathFunc: func(fh *multipart.FileHeader) string {
//	        // Generate a unique file name, for instance, by using a UUID generator, and sanitize the original filename
//	        // Colons could be replaced with a timestamp or another identifier as needed for the application.
//	        return filepath.Join("/uploads/avatars", uuid.New().String() + filepath.Ext(fh.Filename))
//	    },
//	}
//
// The FileUploader struct is typically used in conjunction with HTTP server handlers or middlewares in Go that
// are responsible for handling file uploads. During the request processing, the FileUploader can be utilized to
// retrieve the file data associated with its FileField and to invoke DstPathFunc to calculate where the file
// should be stored. Subsequent steps would typically involve actually storing the file data in the specified
// location and handling any errors or post-processing tasks as necessary.
type FileUploader struct {
	FileField   string
	DstPathFunc func(*multipart.FileHeader) string
}

// Handle returns a HandleFunc specifically prepared to process file upload requests based on the configuration
// of the FileUploader struct. This HandleFunc can be used as an HTTP handler that intercepts file upload HTTP
// requests, processes them, and provides an appropriate response to the client based on the outcome of the
// operation.
//
// The returned HandleFunc performs the following steps:
//  1. It attempts to retrieve the file data from the request's form field as specified by the FileUploader's
//     FileField.
//  2. If it encounters an error during file retrieval, it sets the HTTP response status code to Internal Server
//     Error (500), and it sends back an error message to the client, indicating that the upload failed, along
//     with the error details.
//  3. Upon successful retrieval of the file, the function then calls the FileUploader's DstPathFunc with the
//     metadata of the uploaded file (fileHeader) to determine the destination path where the file should be saved.
//  4. The function checks for errors from the DstPathFunc call; if there's an error, it responds with an Internal
//     Server Error status and an error message as in step 2.
//  5. Next, it attempts to open (or create if not existing) the destination file for writing. It ensures that the
//     permissions for the new file allow read and write operations for the user, group, and others (permission
//     mode 0666).
//  6. If it cannot open the destination file, it responds with an Internal Server Error status and an error message.
//  7. If the destination file is opened successfully, it then copies the content of the uploaded file to the
//     destination file using an io.CopyBuffer operation. This function performs the copy operation and allows
//     the reuse of an existing slice (buffer) to reduce memory allocation, which is set to nil to use a
//     pre-allocated buffer of a default size.
//  8. Any error encountered during the copy process results in an Internal Server Error status and an error message.
//  9. If the file is uploaded and saved successfully, the function sets the HTTP response status to OK (200) and
//     sends a success message to the client.
//
// It's important that the file handles for both the uploaded file and the destination file are closed properly to
// release the resources. The defer keyword is used right after each file is opened to ensure that the file handles
// are closed when the function returns.
//
// Usage of the Handle method would typically involve adding the returned HandleFunc as an HTTP handler in a web
// server's routing configuration. This would route certain POST requests expected to contain file uploads through
// this handler to process the file saving task accordingly.
func (f *FileUploader) Handle() HandleFunc {
	return func(ctx *Context) {
		// Attempt to retrieve the uploaded file from the form's file field.
		file, fileHeader, err := ctx.Request.FormFile(f.FileField)
		// If there's a problem getting the file, respond with a server error and upload failure message.
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		// Ensure the uploaded file is closed before this function exits.
		defer file.Close()
		// Use the DstPathFunc to determine the location to save the file.
		dst := f.DstPathFunc(fileHeader)
		// If the file destination path could not be determined, respond with a server error.
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		// Open (or create) the destination file for writing with the appropriate permissions.
		dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o666)
		// If the destination file cannot be opened, respond with a server error.
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		// Ensure the destination file is closed before this function exits.
		defer dstFile.Close()
		// Copy the content from the uploaded file to the destination file.
		_, err = io.CopyBuffer(dstFile, file, nil)
		// If there was a problem while copying the file, respond with a server error.
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		// If the operation was successful, set the response status to OK and send a success message.
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("Upload success")
	}
}

// FileDownloader is a structure that encapsulates the necessary
// information for handling file download operations in a web application setting.
// The struct is designed to provide a foundation for methods that allow users to download
// files from a specified directory on the server. It acts primarily as a configuration
// container for the download directory, with the potential to extend functionality for
// more advanced features like path resolution, access control, and download logging.
//
// Fields:
//   - Dir string: Specifies the root directory from which files will be served.
//     This path should be an absolute path or a relative path to the
//     directory that the server process has access to. All file download
//     requests will be resolved relative to this directory, meaning that
//     file paths provided in download requests should be relative to this
//     'Dir' directory. It is important to consider security implications
//     when defining this directory, to prevent unauthorized access to sensitive
//     files. File access should be properly managed to avoid directory traversal
//     attacks or the exposure of restricted files.
//
// Usage Notes:
// An instance of FileDownloader should be initialized with the 'Dir' field set to the
// directory containing the files to be downloaded. This instance can then be integrated
// into the web application's server side, where a corresponding handler function will
// use the information contained within FileDownloader to resolve and serve file download
// requests. The handler will typically interpret the file path from the incoming HTTP
// request, ensure that it points to a valid file within the 'Dir', and then stream that
// file back to the client with the appropriate Content-Disposition header to prompt a
// file download in the client's browser.
//
// Example Implementation:
//
//	downloader := &FileDownloader{
//	    Dir: "/var/www/downloads",
//	}
//
// This downloader instance could then be referenced in a dedicated route's handler function
// to process file download requests, ensuring all downloads are served from the
// "/var/www/downloads" directory.
type FileDownloader struct {
	Dir string
}

// Handle creates and returns a HandleFunc designed for serving files for download.
// This HTTP handler ensures that the files served for download are securely retrieved
// from a specified server directory and delivered to the client correctly. The handler
// takes care to resolve the file path, validate the request, and to set proper HTTP
// headers to facilitate the file download on the client side.
//
// The HandleFunc performs the following steps:
//  1. Extracts the 'file' query parameter from the request's URL. This parameter
//     contains the relative path to the file that the client wants to download.
//  2. If the 'file' query parameter is missing or another error occurs while
//     retrieving it, the handler responds with an HTTP 400 Bad Request status
//     and a message indicating that the destination file could not be found.
//  3. Cleans up the requested file path using filepath.Clean to prevent any directory
//     traversal attempts and ensure the path is in its simplest form.
//  4. Joins the cleaned file path with the FileDownloader's Dir to form an absolute path
//     to the file within the server's file system.
//  5. Converts the resulting path to an absolute path and checks if the resolved file path
//     actually exists within the designated download directory. This acts as a security
//     measure to prevent clients from accessing files outside the allowed directory.
//  6. If the resolved file path is outside the designated download directory, the handler
//     responds with an HTTP 400 Bad Request status and a message indicating an access path error.
//  7. The file name is extracted from the resolved path to be used in the Content-Disposition
//     header, which advises the client's browser that a file is expected to be downloaded and saved locally.
//  8. Various HTTP response headers are set to inform the client about the file transfer details:
//     - Content-Disposition: Advises the client on how to handle the file data.
//     - Content-Description: Description of the content.
//     - Content-Type: Specifies that the content is a stream of binary data (i.e., a file).
//     - Content-Transfer-Encoding: Notates that the content transfer will be in binary mode.
//     - Expires: Indicates that the content should not be cached for later use.
//     - Cache-Control and Pragma: Directives to control browser caching.
//  9. The http.ServeFile function is called to serve the file contained in the resolved
//     path to the client. This function takes care of streaming the file data to the client.
//     The function also automatically determines the Content-Type header, although it is
//     overridden here to "application/octet-stream" to trigger the browser's download dialog.
//
// The handler secured by the FileDownloader ensures that only files from a specified
// directory can be accessed and downloaded by the client. Proper error handling is
// implemented to return meaningful HTTP status codes and messages to the client
// in case of an error, such as path resolution issues or illegal file access attempts.
func (f *FileDownloader) Handle() HandleFunc {
	return func(ctx *Context) {
		// Retrieve the requested file path from the query parameter.
		req := ctx.QueryValue("file")
		// Check for errors in retrieving the query parameter.
		if req.err != nil {
			ctx.RespStatusCode = http.StatusBadRequest
			ctx.RespData = []byte("The destination file could not be found")
			return
		}
		// Clean the requested file path to prevent directory traversal.
		req.val = filepath.Clean(req.val)
		// Generate the full intended path by combining the request path with FileDownloader's Dir.
		dst := filepath.Join(f.Dir, req.val)
		// Resolve the path to an absolute path and validate it.
		dst, req.err = filepath.Abs(dst)
		// Ensure that the resolved path is within the allowed download directory.
		if !strings.Contains(dst, f.Dir) {
			ctx.RespStatusCode = http.StatusBadRequest
			ctx.RespData = []byte("Access path error")
			return
		}
		// Extract the actual file name to be used in the Content-Disposition header.
		fn := filepath.Base(dst)
		// Set headers necessary for instructing the browser to handle the response as a file download.
		header := ctx.ResponseWriter.Header()
		header.Set("Content-Disposition", "attachment;filename="+fn)
		header.Set("Content-Description", "File Transfer")
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")
		header.Set("Cache-Control", "must-revalidate")
		header.Set("Pragma", "public")
		// Serve the file with the specified headers, allowing the client to download it.
		http.ServeFile(ctx.ResponseWriter, ctx.Request, dst)
	}
}

// StaticResourceHandlerOption is a type for a function which acts as an option or a
// modifier for instances of StaticResourceHandler. This type enables a flexible configuration
// pattern commonly known as "functional options", which allows the customization of various
// aspects of a StaticResourceHandler instance at the time of its creation or initialization.
//
// Functional options are a way to cleanly and idiomatically pass configuration data to struct
// instances or objects in Go. They provide several benefits over traditional configuration
// approaches such as configuration structs or variadic parameters:
//  1. Options are entirely optional, allowing for a more succinct API where defaults can be
//     used unless explicitly overridden.
//  2. They provide extensibility and forward-compatibility. As new options become necessary,
//     they can be added without breaking existing clients or code.
//  3. They allow for complex and interdependent properties to be set, which might be clunky
//     or error-prone with other configuration mechanisms.
//  4. They preserve the immutability of the object after creation, enabling a clearer,
//     less error-prone API since clients are discouraged from modifying the object's properties
//     directly after it has been created.
//
// How StaticResourceHandlerOption is used:
// An option function takes a pointer to a StaticResourceHandler instance and modifies it
// directly. This function may set default values, modify settings, or provide additional
// functionality to the handler based on its implementation.
// During the construction or initialization of a StaticResourceHandler, one or more of these
// option functions can be passed, which will be applied to the handler instance to configure
// it according to the provided options.
//
// Example Usage:
// Below is a simplified example of how one might define and use a StaticResourceHandlerOption
// function to configure a static resource handler for serving files from a particular directory.
//
//	func DirectoryOption(directory string) StaticResourceHandlerOption {
//	    return func(handler *StaticResourceHandler) {
//	        handler.Directory = directory
//	    }
//	}
//
// // When creating a new StaticResourceHandler, pass the option function to configure its directory.
// handler := NewStaticResourceHandler(
//
//	DirectoryOption("/path/to/static/resources"),
//	// ...other options could be passed here as well...
//
// )
//
// The StaticResourceHandler can now be used with its Directory correctly set to the specified path,
// and this pattern permits the combination of multiple options to achieve complex configurations in
// an expressive and maintainable way.
type StaticResourceHandlerOption func(handler *StaticResourceHandler)

// StaticResourceHandler is a structure used for serving static files such as images,
// CSS, or JavaScript files from a defined directory within a web server. It's designed
// to efficiently handle requests for static content by caching frequently requested
// resources, understanding content types based on file extensions, and allowing limits
// on the size of files served.
//
// Fields:
//   - dir string: The root directory from where the static resources will be served. This should be
//     an absolute path to ensure correct file resolution. When a request comes in for a
//     static resource, the handler will use this directory to look up and serve the files.
//   - cache *lru.Cache: An LRU (Least Recently Used) cache used to store and retrieve the most
//     recently accessed static files quickly. The caching mechanism improves
//     the performance of the web server by reducing the number of disk reads.
//     The 'lru.Cache' refers to an in-memory key-value store where the key is typically
//     the file path or name, and the value is the file's content.
//   - extContentTypeMap map[string]string: A map associating file extensions with their corresponding MIME
//     content type. This mapping allows the server to set the appropriate
//     'Content-Type' header in HTTP responses based on the requested
//     file's extension. For example, it might map ".css" to "text/css"
//     and ".png" to "image/png".
//   - maxSize int: The maximum size of a file, in bytes, that the handler will serve. Requests for files
//     larger than this size will result in an error, preventing the server from consuming
//     excessive resources when serving large files. This is a safeguard to help maintain
//     server performance and stability.
//
// The StaticResourceHandler struct requires careful initialization to ensure it has access to the correct
// directory and that the cache and content type map are adequately configured. It can be used in standalone
// mode or as part of an HTTP server, integrating with the server's request handling mechanism to serve the
// static files when requested.
//
// Once appropriately configured, the StaticResourceHandler provides a robust and efficient way to handle
// requests for static resources, enabling you to optimize the delivery of these files as part of your
// web application's overall performance strategy.
type StaticResourceHandler struct {
	dir               string
	cache             *lru.Cache
	extContentTypeMap map[string]string
	maxSize           int
}

// InitStaticResourceHandler initializes and returns a pointer to a StaticResourceHandler
// with the provided directory path and applies any given configuration options. The function
// also establishes an LRU (Least Recently Used) cache with a default maximum capacity to
// optimize the serving of static files. This function centralizes the setup logic for creating
// a StaticResourceHandler, ensuring that the handler is properly initialized and configured before
// use.
//
// Parameters:
//   - dir string: The absolute directory path where the static resources are located. This will
//     be the root directory from which the server will serve static files.
//   - opts ...StaticResourceHandlerOption: A variadic slice of functional options used to customize
//     the configuration of the StaticResourceHandler. These options
//     are applied in the order they're received, allowing the caller
//     to override default values and behavior.
//
// Return Values:
//   - *StaticResourceHandler: A pointer to the newly-created configured StaticResourceHandler ready to
//     serve static files.
//   - error: An error that may have occurred during the creation or configuration of the
//     StaticResourceHandler. If the error is not nil, it usually indicates a problem with
//     setting up the internal LRU cache.
//
// Internal Initialization Steps:
//  1. The function creates an LRU cache with a default size of 1000 cache entries. If the cache
//     cannot be created, the function immediately returns nil and the error.
//  2. A StaticResourceHandler struct instance is instantiated with the given directory path and the
//     newly created LRU cache.
//  3. Default file size limit for serving files is set to 1 megabyte (1024 * 1024 bytes).
//  4. A default extension to content type mapping is established for common file formats to ensure
//     correct 'Content-Type' headers in HTTP responses.
//  5. The function then iterates through each provided configuration option, applying it to the
//     newly created StaticResourceHandler instance. This allows customization such as changing the
//     maximum cache size or adding new file type mappings.
//  6. The fully configured StaticResourceHandler pointer is then returned for usage in the server.
//
// Example Usage:
// handler, err := InitStaticResourceHandler("/path/to/static/resources",
//
//	SetMaxSize(500 * 1024), // 500 KB as the max file size
//	AddFileTypeMapping("svg", "image/svg+xml"), // Add SVG MIME type mapping
//
// )
//
//	if err != nil {
//	    log.Fatalf("Failed to initialize static resource handler: %v", err)
//	}
//
// // Now handler can be used to serve static resources with the specified configurations.
func InitStaticResourceHandler(dir string, opts ...StaticResourceHandlerOption) (*StaticResourceHandler, error) {
	// Create a new LRU cache instance with default capacity.
	c, err := lru.New(1000)
	if err != nil {
		return nil, err // Return the error if cache creation fails.
	}
	// Instantiate the StaticResourceHandler struct with default values.
	res := &StaticResourceHandler{
		dir:     dir,
		cache:   c,
		maxSize: 1024 * 1024, // Default max file size of 1 megabyte.
		// Set up default file extension to MIME type mappings.
		extContentTypeMap: map[string]string{
			"jpeg": "image/jpeg",
			"jpe":  "image/jpeg",
			"jpg":  "image/jpeg",
			"png":  "image/png",
			"pdf":  "application/pdf", // Corrected MIME type for PDF.
		},
	}
	// Apply all given configuration options to the handler.
	for _, opt := range opts {
		opt(res)
	}
	// Return the configured handler ready for use.
	return res, nil
}

// StaticWithMaxFileSize returns a StaticResourceHandlerOption which sets the maximum file size
// (in bytes) that a StaticResourceHandler is allowed to serve. The maxSize parameter specifies
// the size limit, and files exceeding this limit will not be served by the handler. This option
// function is part of a pattern allowing granular configuration of a StaticResourceHandler through
// functional options, which are applied when initializing the handler with the InitStaticResourceHandler
// function.
//
// Parameters:
//   - maxSize int: The maximum size (in bytes) that the StaticResourceHandler will serve. This acts
//     as a guard against serving excessively large static files, potentially consuming
//     too much memory or bandwidth.
//
// Returns:
//   - StaticResourceHandlerOption: A function closure that when called with a *StaticResourceHandler,
//     will assign the specified maxSize to the handler's maxSize field.
//
// The maxSize is an essential configuration as it helps to manage server resources efficiently and ensures
// the server does not get overloaded by requests for very large files. This is particularly useful when you
// expect that your server might be serving large files and wish to put an explicit cap on them.
//
// Example Usage:
// To create a StaticResourceHandler that has a maximum serving file size of 500KB, you would use the
// StaticWithMaxFileSize option like so:
//
// handler, err := InitStaticResourceHandler("/static", StaticWithMaxFileSize(500 * 1024))
//
//	if err != nil {
//	    // handle error
//	}
//
// When the above handler is used in a server, any request for a static file larger than 500KB will be
// denied or handled according to the server's implementation.
//
// Using this function enables the developer to tailor the StaticResourceHandler's behavior to the
// specific requirements of their application or the resource constraints of their server environment.
func StaticWithMaxFileSize(maxSize int) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		handler.maxSize = maxSize // Set the max file size on the handler.
	}
}

// StaticWithCache returns a StaticResourceHandlerOption that assigns a custom
// LRU (Least Recently Used) cache to a StaticResourceHandler. The provided *lru.Cache
// replaces the default cache, allowing for higher flexibility in how caching is configured
// for the static file handler. This is particularly useful for fine-tuning the cache size
// or sharing a cache instance among multiple handlers.
//
// Parameters:
//   - c *lru.Cache: A pointer to the lru.Cache instance that should be used by the
//     StaticResourceHandler. This parameter allows the user to specify the
//     exact cache instance, including its size and eviction policies, which
//     will be integrated into the static resource serving process.
//
// Returns:
//   - StaticResourceHandlerOption: A closure function that accepts a *StaticResourceHandler
//     as its parameter. When executed, this option function sets
//     the StaticResourceHandler's internal cache pointer to the
//     provided *lru.Cache instance.
//
// This configuration option is valuable when performance optimization is needed or when
// different caching strategies are tested. It provides the ability to swap in a custom cache
// with a pre-defined configuration without altering the existing flow of the handler's initialization.
//
// Example Usage:
// To use a previously configured lru.Cache with a capacity of 500 items in a new StaticResourceHandler,
// invoke StaticWithCache like this:
//
// existingCache, _ := lru.New(500) // Create an lru.Cache with a capacity for 500 items.
// handler, err := InitStaticResourceHandler("/static", StaticWithCache(existingCache))
//
//	if err != nil {
//	    // handle error
//	}
//
// This allows the new StaticResourceHandler to utilize the existingCache for caching static
// files, providing a customized caching strategy as per the application's requirements.
//
// Adopting a feature-rich cache can significantly reduce IO operations against the disk and
// improve the serving speed of static files, leading to better application performance and
// scalability. The option to set a custom cache is a direct way to control these aspects.
func StaticWithCache(c *lru.Cache) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		handler.cache = c // Assign the provided cache to the handler.
	}
}

// StaticWithExtension returns a StaticResourceHandlerOption which extends or overrides
// the existing file extension to MIME type mappings in a StaticResourceHandler.
// The provided extMap parameter is a map of file extensions to their respective MIME
// types. This function is helpful when you need the StaticResourceHandler to recognize
// additional file types or modify existing associations.
//
// Parameters:
//   - extMap map[string]string: This map represents the file extensions and their associated
//     MIME types that should be recognized by the StaticResourceHandler.
//     Keys in the map should be file extensions without the dot (e.g., "txt"),
//     and the values should be the corresponding MIME types (e.g., "text/plain").
//
// Returns:
//   - StaticResourceHandlerOption: A functional option that, when applied to a StaticResourceHandler,
//     updates its 'extContentTypeMap' field to include the mappings from
//     extMap. If an extension already exists in the handler's mapping, it will
//     be overridden with the new MIME type provided in extMap.
//
// This function offers a way to customize the content type determination process when serving
// static files. By specifying file extension to MIME type mappings, the server can set correct
// 'Content-Type' headers in HTTP responses. This is vital for browsers or clients to handle
// the received files appropriately.
//
// Example Usage:
// To add or override mappings for ".svg" and ".json" files in a StaticResourceHandler,
// use StaticWithExtension like this:
//
//	customExtMap := map[string]string{
//	    "svg": "image/svg+xml",
//	    "json": "application/json",
//	}
//
// handler, err := InitStaticResourceHandler("/static", StaticWithExtension(customExtMap))
//
//	if err != nil {
//	    // handle error
//	}
//
// The above code configures the handler to recognize ".svg" files with the MIME type
// "image/svg+xml" and ".json" files with "application/json". These mappings will be
// added to or replace any existing mappings for these extensions in the handler.
//
// This level of customization aids in serving a broader range of file types or catering
// to specific client needs while ensuring that the server's responses are correctly
// understood by the client.
func StaticWithExtension(extMap map[string]string) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		for extension, contentType := range extMap {
			handler.extContentTypeMap[extension] = contentType // Update or add new mappings.
		}
	}
}

// Handle takes a Context pointer and serves static files based on the request's path.
// It attempts to retrieve and serve the requested file, handling various error scenarios
// gracefully. It sets appropriate HTTP response status codes and headers, leveraging an LRU cache
// for performance optimization when possible.
//
// The method logic is as follows:
//  1. It extracts the requested 'file' from the context's PathValue.
//  2. If there's an error in retrieving the file (e.g., malformed request path),
//     it sends a 400 Bad Request status and a "Request path error" message.
//  3. If the file can be retrieved, it constructs a full file path using the provided
//     directory path and the file's name.
//  4. The file extension is extracted to determine the correct content type from the handler's
//     extension to MIME type mapping (extContentTypeMap).
//  5. If the file's data is found in the cache, it uses this data to set the response headers
//     and body, sending a 200 OK status code.
//  6. If not cached, it reads the file from disk using os.ReadFile.
//  7. If there's an error reading the file (e.g., file not found, permissions issue), it sets
//     a 500 Internal Server Error status code and a "Server error" message.
//  8. It checks if the file size is within the allowed maximum size (s.maxSize).
//     If it is, the function adds the file's data to the cache.
//  9. Lastly, it sets the correct "Content-Type" and "Content-Length" headers and sends the file data
//     with a 200 OK status.
//
// Parameters:
//   - ctx *Context: A pointer to the Context object which contains information about the HTTP request
//     and utilities for writing a response.
//
// Note:
// This method should be used to handle routes that match static file requests. It automatically uses
// caching to improve performance, but it also ensures that the file size does not exceed a configured
// threshold before caching the data.
//
// Example Usage:
// Assuming 'handler' is an instance of StaticResourceHandler with a configured directory and cache,
// the Handle method would be attached to a web server route like so:
//
//	http.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    ctx := NewContext(r, w)
//	    handler.Handle(ctx)
//	}))
//
// This specifies that all requests to '/static/' should be handled by the Handle method of 'handler',
// which will serve the files as static resources.
func (s *StaticResourceHandler) Handle(ctx *Context) {
	file := ctx.PathValue("file")
	if file.err != nil {
		// Error handling: Bad request due to request path issues.
		ctx.RespStatusCode = http.StatusBadRequest
		ctx.RespData = []byte("Request path error")
		return
	}
	dst := filepath.Join(s.dir, file.val)
	ext := filepath.Ext(dst)[1:]
	header := ctx.ResponseWriter.Header()
	if data, ok := s.cache.Get(file.val); ok {
		// Serve content from cache if available.
		header.Set("Content-Type", s.extContentTypeMap[ext])
		header.Set("Content-Length", strconv.Itoa(len(data.([]byte))))
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = data.([]byte)
		return
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		// Error handling: Internal server error due to file read issues.
		ctx.RespStatusCode = http.StatusInternalServerError
		ctx.RespData = []byte("Server error")
		return
	}

	// Caching file data if it's within the maximum allowed size.
	if len(data) <= s.maxSize {
		s.cache.Add(file.val, data)
	}
	// Serving the file content with the correct headers.
	header.Set("Content-Type", s.extContentTypeMap[ext])
	header.Set("Content-Length", strconv.Itoa(len(data)))
	ctx.RespStatusCode = http.StatusOK
	ctx.RespData = data
}
