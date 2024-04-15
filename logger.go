package mist

// Logger is an interface that specifies logging functionality.
// The Logger interface declares one method, Fatalln, which is responsible
// for logging critical messages that will lead to program termination.
//
// The Fatalln method takes a mandatory string message (msg) as the first
// parameter, followed by a variadic set of arguments (args). The variadic
// part means that the method can accept any number of arguments following
// the first string message. This allows for formatting and inclusion of
// various data types within the logged message.
//
// Upon invocation, Fatalln will log the provided message along with any
// additional provided arguments. After logging the message, Fatalln will
// call os.Exit(1) to terminate the program with a status code of 1,
// which indicates an error state. This method is typically used to log
// unrecoverable errors where continuing program execution is not advisable.
//
// It is important to use Fatalln cautiously as it will halt the execution
// flow immediately after logging, which can cause defer statements and
// resource cleanups to be bypassed.
type Logger interface {
	Fatalln(msg string, args ...any)
}

// defaultLogger is a variable of type Logger, which serves as the default
// logging instance used throughout the application. As an interface, Logger
// abstracts the details of the logging implementation, allowing for flexibility
// in the underlying logging mechanism used.
//
// The purpose of having a defaultLogger is to provide a central, commonly
// accessible logging facility, so that different parts of the application can
// log messages, warnings, and errors in a consistent manner. It ensures that
// all logging activities are unified and can be easily configured or redirected
// from a single point.
//
// Before using defaultLogger, it must be initialized with an actual implementation
// of the Logger interface. This initialization process typically occurs during
// the application's startup phase, where a specific logging implementation (such
// as logrus, zap, or a custom logger) is instantiated and assigned to defaultLogger.
// This allows the application to record logs according to the configured logging
// level (e.g., INFO, WARN, ERROR), format (e.g., JSON, plaintext), and destination
// (e.g., console, file, remote logging server).
//
// The specific logging implementation used can be swapped out with minimal changes
// to the rest of the application, thanks to the abstraction provided by the Logger
// interface. This design enhances the maintainability and scalability of the logging
// system within the application.
var defaultLogger Logger

// SetDefaultLogger is a function that allows for the configuration of the
// application's default logging behavior by setting the provided logger
// as the new default logger.
//
// Parameters:
//
//	log Logger: This parameter is an implementation of the Logger interface.
//	            It represents the logger instance that the application should
//	            use as its new default logger. This logger will be used for
//	            all logging activities across the application, enabling a
//	            consistent logging approach.
//
// Purpose:
// The primary purpose of SetDefaultLogger is to provide a mechanism for
// changing the logging implementation used by an application at runtime.
// This is particularly useful in scenarios where the logging requirements
// change based on the environment the application is running in (e.g.,
// development, staging, production) or when integrating with different
// third-party logging services.
//
// Usage:
// To use SetDefaultLogger, an instance of a Logger implementation needs to
// be passed to it. This can be a custom logger tailored to the application's
// specific needs or an instance from a third-party logging library that
// adheres to the Logger interface. Once SetDefaultLogger is called with
// the new logger, all subsequent calls to the defaultLogger variable
// throughout the application will use this new logger instance,
// thereby affecting how logs are recorded and stored.
//
// Example:
// Suppose you have an application that uses a basic logging mechanism by
// default but requires integration with a more sophisticated logging
// system (like logrus or zap) for production environments. You can
// initialize the desired logger and pass it to SetDefaultLogger during
// the application's initialization phase. This ensures that all logging
// throughout the application uses the newly specified logger.
//
// Note:
// It is important to call SetDefaultLogger before any logging activity occurs
// to ensure that logs are consistently handled by the chosen logger. Failure
// to do so may result in some logs being handled by a different logger than
// intended, leading to inconsistency in log handling and potential loss of
// log data.
func SetDefaultLogger(log Logger) {
	defaultLogger = log
}
