# Specifications for the Feature Automated Library Scan

I want to build a media sever that serves video in our home network. This describe the feature how the library scan should get invoked at start-up and periodically during run to automatically scan for new files.

# Basic idea

The scan is currently implemented via an HTTP handler. Hence, it always needs to be invoked via a POST. This is not very convenient. I want to start the scan at each server startup and also periodically.

# Requirements

- The server scan should be started at each server startup
- The server scan should be performed periodically based on a configurable duration. E.g. every 30 sec

# Future features

Additional feature which will have an impact on this feature will be implemented later. They are not yet to consider, but they might provide hints in terms of design and architecture so that they will be more easily implementable.

- Additional jobs that might necessary to be run on startup

# Tasks

- Analyze the requirements
- Check them for contradictions
- Check them for completeness
- Create a comprehensive design how this feature will be implemented
- Implement the feature including tests
- Always ask missing information

# Constrains

- Use the existing structure of the code when designing your approach
- Use the existing way of initializing components in the app package
- Always write unit tests and where appropriate also integration tests
- use bin as the directory to startup the server and to create the necessary directories based on the config to avoid cluttering other directories
