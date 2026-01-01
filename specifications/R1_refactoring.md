# Specifications for Regular Refactoring

I want to build a media sever that serves video in our home network. This describe the regular task of checking for simplifications and refactoring.

# Basic idea

Code bases tend to bloat over time and we need a regular review to check what we cloud simplify and refactor.

# Tasks

- Check the existing implementation for code smells that need refactoring
- Check for overly complex or poorly structured methods that need simplification and refactoring
- Check for overly complex or poorly structured files that need simplification and refactoring
- Check for overly complex or poorly structured packages that need simplification and refactoring
- Are there methods we do not need any more
- Can we improve the naming of variables, structs, methods, files e.t.c
- Should we improve our package structure
- Are there further improvements that we should examine

# Constrains

- Check if there are any uncommitted files. If so, stop and don't continue! This ensures that we start from a committed base and that we can always revert back in case we have to.
- Keep the architecture, design and coding style of our current project if you use code from the old server
- Always ask before you want to implement an improvement
- Use the existing structure of the code when designing your approach
- Use the existing way of initializing components in the app package
- Always write unit tests and where appropriate also integration tests
- use bin as the directory to startup the server and to create the necessary directories based on the config to avoid cluttering other directories
- This is a development project. We don't have running servers. So all breaking changes are fine and impose no issues.
