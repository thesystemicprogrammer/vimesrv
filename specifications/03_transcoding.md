# Specifications for the Feature Transcoding

I want to build a media sever that serves video in our home network. This describe the feature how the newly added files should be transcoded so that they can be streamed via DASH and HLS

# Basic idea

After a new video file is added to the library, the next jobs should start to transcode the video file an make it available via DASH and HSL including all audio stream and subtitles included in the video. Use as base the old implementation in ./old_srv/vimesrv. This implementation is currently broken but the transcoding worked very well. But adapt it to the architecture and coding style of this new project.

# Requirements

- After a new file is added to the library, start the transcoding mechanisms
- Create resolutions according to the active presets defined in the config
- Transcode all audio stream. Save the audio stream only once for all preset resolutions to save space
- Extract all subtitles. Save them only once for all preset resolutions to save space.
- Provide endpoints to serve a DASH manifest and a HSL master playlist so we can support both specifications.
-

# Future features

Additional feature which will have an impact on this feature will be implemented later. They are not yet to consider, but they might provide hints in terms of design and architecture so that they will be more easily implementable.

- Enrich the media file with external data from TMDB or similiar

# Tasks

- Analyze the requirements
- Check them for contradictions
- Check them for completeness
- Create a comprehensive design how this feature will be implemented
- Implement the feature including tests
- Always ask missing information

# Constrains

- Use the old implementation under ./old_srv/vimesrv and copy it's logic
- The job system is already implemented in the new server. Use this job system for the transcoding jobs
- Keep the architecture, design and coding style of our current project if you use code from the old server
- Always ask if you see improvements compared to the old code
- Use the existing structure of the code when designing your approach
- Use the existing way of initializing components in the app package
- Always write unit tests and where appropriate also integration tests
- use bin as the directory to startup the server and to create the necessary directories based on the config to avoid cluttering other directories
- This is a development project. We don't have running servers. So all breaking changes are fine and impose no issues.
