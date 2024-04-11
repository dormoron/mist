package mist

import (
	"bytes"
	"context"
	"html/template"
	"io/fs"
)

// TemplateEngine defines the contract for a template rendering system used to generate text output
// (such as HTML, XML, etc.) based on predefined templates and dynamic data. Such an engine typically
// supports a variety of operations related to template parsing, data binding, and the generation of
// rendered output. This interface abstracts these operations so that different implementations of
// templating engines can be used interchangeably within an application.
//
// The interface includes a single method, Render, which must be implemented by any concrete type that
// purports to be a template engine. This method takes a template name and associated data and produces
// the rendered output as a slice of bytes, along with an error if the rendering process fails.
//
// Render method parameters:
//   - ctx (context.Context): The context parameter allows the Render method to be aware of the broader
//     context of the application, such as deadlines, cancellation signals, and
//     other request-scoped values. It enables the rendering process to handle
//     timeouts or cancellations as per the application's context.
//   - templateName (string): This is the identifier or the name of the template that needs to be rendered.
//     The templating engine uses this name to look up and load the appropriate
//     template file or content from an internal cache or the filesystem.
//   - data (any): The data parameter is an interface{} type, allowing any kind of Go value to be passed
//     as the data context for the template. This data is then utilized by the template engine
//     to dynamically populate placeholders within the template, enabling the generation of
//     customized output based on the data provided.
//
// Render method returns:
//   - []byte: When successful, Render returns the generated content as a byte slice, which can be used
//     directly as output to a response writer or converted to a string for further processing.
//   - error: If any errors occur during the rendering of the template, Render returns an error detailing
//     the issue encountered. This could be due to reasons like template parsing errors, missing data
//     required to fill in the template, file system issues when accessing template files, etc.
//
// Notes:
//   - Implementations of the TemplateEngine interface should ensure to handle any template-specific syntax
//     and errors internally, providing a consistent and easy-to-use API for rendering content within an
//     application.
//   - The flexibility of this interface allows for custom implementations that could include enhanced
//     features like caching, internationalization support, and more.
type TemplateEngine interface {
	// Render takes the name of a template and the data object to be used in rendering, and returns the
	// rendered template as a slice of bytes, or an error if the rendering could not be completed. This
	// method abstracts the rendering logic, enabling different templating systems to be integrated into
	// the application as needed.
	Render(ctx context.Context, templateName string, data any) ([]byte, error)
}

// GoTemplateEngine is a struct that adheres to the TemplateEngine interface. It represents a concrete
// implementation using Go's standard library "text/template" or "html/template" for rendering templates.
// This structure enables the application to perform template rendering operations using Go's built-in
// template parsing and execution capabilities.
//
// The GoTemplateEngine struct contains a single field:
//   - T: This field holds a pointer to a template.Template object from Go's template package. The
//     template.Template type is a thread-safe, compiled representation of a set of templates that
//     can be executed with particular data inputs to produce rendered content. This field should
//     be pre-populated with parsed templates upon the creation of a GoTemplateEngine instance.
//
// When initializing a new GoTemplateEngine, the typical process includes parsing the template files
// or strings using the Parse, ParseFiles, or ParseGlob functions provided by Go's template package.
// These functions compile the templates and return a *template.Template object, which is then assigned
// to the T field.
//
// Using the T field, the GoTemplateEngine can execute the named templates with given data using the
// ExecuteTemplate method, which satisfies the Render method of the TemplateEngine interface. The
// execution process replaces template tags with corresponding data and generates the final rendered
// output.
//
// Example of initializing a GoTemplateEngine with parsed templates could look like this:
//
//	func NewEngine() (*GoTemplateEngine, error) {
//	    tmpl, err := template.ParseFiles("templates/base.html", "templates/index.html")
//	    if err != nil {
//	        return nil, err
//	    }
//	    return &GoTemplateEngine{T: tmpl}, nil
//	}
type GoTemplateEngine struct {
	// T is a pointer to a compiled template object from the Go standard library's template package.
	// It stores all the parsed template files that can be executed to generate output based on dynamic data.
	T *template.Template
}

// Render is the method that implements the rendering functionality for the GoTemplateEngine. It
// satisfies the TemplateEngine interface's Render method, allowing the GoTemplateEngine to be used
// wherever a TemplateEngine is required. This method executes the named template using the provided
// data and generates a rendered output as a slice of bytes.
//
// Parameters:
//   - ctx context.Context: This parameter provides context for the render operation. It can carry deadlines,
//     cancellations signals, and other request-scoped values across API boundaries and
//     between processes. In this method, ctx is not directly used but could be utilized
//     by extending the method further to support context-aware operations in the render
//     process, such as logging, tracing, timeout, or cancellation.
//   - templateName string: The name of the template to be executed. This must correspond to the name given to
//     one of the parsed templates contained within the *template.Template associated
//     with the GoTemplateEngine.
//   - data any: The data to be applied to the template during rendering. This could be any type of value
//     that the underlying templates are expected to work with, typically a struct, map, or a
//     primitive type that the template fields can reference.
//
// Returns:
//   - []byte: A byte slice containing the rendered output from the executed template. If rendering is
//     successful without any errors, the content of the slice represents the completed output,
//     ready to be used as needed (e.g., sent as an HTTP response).
//   - error: An error that may have been encountered during the template execution. If such an error
//     occurs, it's usually related to issues like an undefined template name, missing data for
//     placeholders in the template, or execution errors within the template. If the error is
//     non-nil, the byte slice returned will be nil, and the error should be handled appropriately.
//
// The Render method works by creating a new bytes.Buffer to act as an in-memory writer where the
// rendered output is temporarily stored. The ExecuteTemplate method of the *template.Template is then
// called with this buffer, the name of the template, and the data to be used for rendering. If the
// template execution is successful, the content of the buffer is returned as a byte slice. In case
// of an error, the error returned from ExecuteTemplate is passed through to the caller.
func (g *GoTemplateEngine) Render(ctx context.Context, templateName string, data any) ([]byte, error) {
	// Create a new bytes.Buffer to hold the generated content after the template is executed.
	bs := &bytes.Buffer{}
	// Execute the named template, writing the output into the buffer (`bs`). The method passes the given
	// `data` to the template, which fills the placeholders within the template.
	err := g.T.ExecuteTemplate(bs, templateName, data)
	// Return the contents of the buffer as a slice of bytes, and any render error that may have occurred.
	return bs.Bytes(), err
}

// LoadFromGlob is a method of GoTemplateEngine that loads and parses template files based on a specified
// pattern (glob). After execution, the GoTemplateEngine's T field is updated to contain the new parsed
// templates. This method is typically used during initialization or when the templates need to be refreshed.
//
// Parameter:
//   - pattern string: The pattern parameter is a string that specifies the glob pattern used to identify
//     the template files that should be parsed and added to the template set. A glob pattern
//     may include wildcards (e.g., "*.tmpl" for all files with the .tmpl extension in the current
//     directory, or "templates/*.html" for all .html files within a 'templates' directory).
//
// Returns:
//   - error: This function will return an error if the parsing operation fails. The error might be caused by
//     an inability to read files due to incorrect permissions, non-existent files, or syntax errors within
//     the template files themselves. If an error is returned, the T field of the GoTemplateEngine struct
//     is not updated with any new templates.
//
// The method works by invoking template.ParseGlob from Go's template package with the provided pattern
// string. This function reads and parses all the template files that match the pattern into a new template
// set. The parsed templates are then assigned to the T field of the GoTemplateEngine instance.
//
// When calling this method, it's important to handle any errors returned to ensure the application is aware
// if the templates failed to load. A successful execution means that the T field is now ready to execute
// the loaded templates with the Render method.
func (g *GoTemplateEngine) LoadFromGlob(pattern string) error {
	// Declare an error variable to capture any errors from the ParseGlob operation.
	var err error
	// Update the T field of the current GoTemplateEngine instance by parsing templates that match
	// the provided pattern. ParseGlob will read and parse the files, compiling them into a template set.
	g.T, err = template.ParseGlob(pattern)
	// Return any error encountered during the parsing process. If err is nil, parsing was successful.
	return err
}

// LoadFromFiles is a method on the GoTemplateEngine struct that parses template files specified by
// the filenames and updates the GoTemplateEngine's T field with the parsed template set. This method
// allows for explicit control over which template files are loaded and is commonly used for initializing
// the engine with a known set of templates or updating the templates at runtime.
//
// Parameters:
//   - filenames ...string: A variadic parameter that accepts an arbitrary number of strings, each representing
//     the path to a template file that should be included in the parsing process. This allows
//     for a flexible number of files to be loaded at once without needing to specify them in a
//     slice.
//
// Returns:
//   - error: If the method encounters any issues while trying to read or parse the provided template files, it
//     will return an error. Potential issues include file I/O errors, file not found errors, permission
//     denials, or problems within the template files themselves, such as syntax errors. If an error is
//     returned, the T field remains unchanged, and the new templates are not loaded.
//
// The method makes use of the template.ParseFiles function provided by Go's template package. The ParseFiles
// function takes the provided file paths, reads the contents, and attempts to parse them into templates within
// a template set. These templates are then stored in the T field of the GoTemplateEngine.
//
// Error handling after calling this method is essential to ensure the application's stability, as templates are
// fundamental for generating output based on them. On successful execution, the templates become immediately
// available for rendering through the engine's Render method.
func (g *GoTemplateEngine) LoadFromFiles(filenames ...string) error {
	// Initialize an error variable that will hold any errors returned by the ParseFiles function.
	var err error
	// Assign to the T field a new template set consisting of the templates obtained from parsing the files
	// using the filenames provided to the method. The filenames are spread into the function call using the
	// variadic spread operator (...).
	g.T, err = template.ParseFiles(filenames...)
	// Return the error, if any, from the parsing process. A nil error signifies successful loading and parsing
	// of the template files.
	return err
}

// LoadFromFS is a method on the GoTemplateEngine struct that loads and parses templates from a file system
// abstraction (fs.FS) using specified glob patterns. It updates the GoTemplateEngine's T field with the resulting
// template set. This allows the loading of templates from various sources that implement the fs.FS interface, such
// as in-memory file systems, making the method flexible and adaptable to different run-time environments.
//
// Parameters:
//   - fs.FS: The fs.FS interface instance represents an abstract file system to load templates from. It
//     provides a way to abstract the file system operations through methods like Open, making it
//     possible to work with different file systems (e.g., an actual OS file system, in-memory file
//     systems, embedded file systems).
//   - patterns ...string: This variadic parameter takes multiple string arguments, each representing a glob
//     pattern used to match files within the provided file system (fs). These patterns
//     tell the method which files to consider when loading and parsing the templates.
//
// Returns:
//   - error: If there is an issue parsing the templates using the provided file system and patterns, the method
//     will return an error. This error could be a result of file I/O operations issues, files not matching
//     any patterns, permission issues, or problems within the template files such as syntax errors. In the
//     case of an error, the GoTemplateEngine's T field will not be updated with any new templates.
//
// Internally, the method uses the template.ParseFS function from Go's template package. This function is designed
// to work with the fs.FS interface to read and parse templates. The method parses all the files that match the
// provided patterns into a template set, which is then stored in the T field of the current GoTemplateEngine instance.
//
// As with other template loading methods, proper error handling is crucial to ensure the application can handle
// cases where templates could not be loaded. If no errors are returned, the engine can proceed to render these
// templates through its Render method, as they are immediately available for use.
func (g *GoTemplateEngine) LoadFromFS(fs fs.FS, patterns ...string) error {
	// Initialize an error variable to capture and potential errors that ParseFS might return.
	var err error
	// Attempt to parse files from the provided file system (fs) that match the provided glob patterns.
	// The parsed templates are stored in the GoTemplateEngine's T field.
	g.T, err = template.ParseFS(fs, patterns...)
	// Return any errors encountered during the parsing process. A nil error indicates a successful
	// parsing and loading of the templates into the engine.
	return err
}
