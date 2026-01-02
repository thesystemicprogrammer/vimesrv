# Specifications for Potential Improvements

I want to build a media sever that serves video in our home network. This describes potential improvements or code smells I discovered while browsin the code.

# Basic idea

Whenever I discover something odd or something I don't understand, I write it down here in the code smell list. I want you to check and to explain, why something was solved the way it currently is and to evaluate if there are better options.

# Smells

- Transcode Path pattern: Why do I have to set a pattern and not only the path where we want to put the transcode directories and files?
- Setting the language label on the server: What about multi language clients? Should the label not be determined on client side with the client language
- How do we update series metadata? When there is a new season, the metadata changes so we need an update
- While transcoding, the media is already visible in the frontend, but there are no streams. Can we somehow ensure that we cannot try to start playing a stream as long as transcoding is still running
- Translation of fixed front end texts: We can switch language but only for TMDB metadata. But we need to translate the whole frontend
- Translation fetching in TMDB: Is it consistent or could it be improved simpliyfied?
- Translation fetching in TMDB: Is it consistent or could it be improved simpliyfied?
-

# Tasks

- Analyze my list of smells
- Explain why they are implemented that way
- Research if there are better options how this could be implemented

# Constrains

- Check if there are any uncommitted files. If so, stop and don't continue! This ensures that we start from a committed base and that we can always revert back in case we have to.
- Keep the architecture, design and coding style of our current project if you use code from the old server
- Always ask before you want to implement an improvement
- Use the existing structure of the code when designing your approach
- Use the existing way of initializing components in the app package
- Always write unit tests and where appropriate also integration tests
- use bin as the directory to startup the server and to create the necessary directories based on the config to avoid cluttering other directories
- This is a development project. We don't have running servers. So all breaking changes are fine and impose no issues.
