# Specifications for the Feature Library Scan

I want to build a media sever that serves video in our home network. This describe the feature how new files are handled and accepted into the library.

# Basic idea

The idea is that new files can be put into a kind of a staging area (aka directory). When a library scan is invoked, the server checks this directory for new files, which are not already part of the library. If the file does not yet exist in the library, the file is copied into the library area (aka directory) and stored in the media file table of the database.

# Requirements

- The media scan checks the staging are recursively for new video files. That means that the scan directory might also contain a hierarchy of subdirectories
- If a file is found, it is checked with ffprobe if it is a valid video file
- If it is not valid video file, a warning is logged and the file is skipped
- If is is a valid video file, a fingerprint is created. Use a clever algorithm to find a good balance between collision avoiding and speed
- Check with the created fingerprint, if this exact video file already exist in the library
- If it does not exists, create a new entry in the media file table, copy the file from the staging area to the library area (aka directory). Create a new subdirectory that uses the fingerpring as name. Then copy the media file into this subdirectory.
- Afterwards, create a new entry in the media file table. Use the fingerprint as key. Put the file path in canonical form a field into the entity. Additionally, store relevant meta data as well.
- Delete the file from the staging area
- If there is no more content to scan, delete all empty subdirectories. Keep those, where still files are existing (those which were not video files or if the processing stopped with an error)

# Future features

Additional feature which will have an impact on this feature will be implemented later. They are not yet to consider, but they might provide hints in terms of design and architecture so that they will be more easily implementable.

- Metadata fetching: It must be possible to fetch media metadata (e.g. via TMDB) to create a media entity which will link to the library entity. We will need a way to have different status for the library entity: It can be linked to a media entity, it can be not linked to a media entity at all, or we can have multiple possible entries where we are not sure which one is the right one.
- Transcoding: I want to transcode the files so they can be served as HLS or DASH.

# Tasks

- Analyze the requirements
- Check them for contradictions
- Check them for completeness
- Create a comprehensive design how this feature will be implemented
- Implement the feature including tests
- Always ask missing information

# Constrains

- Use the existing structure of the code when designing your approach
- Always write unit tests and where appropriate also integration tests
