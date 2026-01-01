# Specifications for the Feature Metadata Enrichment TMDB

I want to build a media sever that serves video in our home network. This describe the feature on how to enrich our media information with metadata from tmdb

# Basic idea

You are a versed user experience designer and you know how to build compelling video servers & clients. It should be easy and fun to use them. You do a lot of research and learn from the best in the field how to setup things properly. A crucial part is to add metadata (text and images) from platform like TMBD to enhance the user experience.

# Requirements

## Types of media

- I want to be able to store metadata about single movies
- I want to be able to store metadata about series. Here I can store metadata for the whole series, for an season and for a single episode
- The distinction if it is a movie or a series will only become clear when metadata have been selected for a media entity

## Selection of Metadata

- The metadata are searched based on the title of the video
- Develop a clever algorithm how based on the file name of the video we can fetch the correct metadata in TMDB
- Series will have the season and the episode in their name with different patterns (\_S01_E02, -S01-E02, S01E02, ...)
- if the confidence level of the algorithm is high enough, the type of the video (movie or series) is determined and the metadata are fetched
- if it is a series, then fetch metadata for the whole series and for the season as well, if they do not already exist
- if you have multiple possible selection but you are unsure which one is the right one, prepare it for the user to select it in the pwa player app
- if there are no metadata mark it with a status so we can give a hint to the user in the pwa player app that a manual search is necessary

## Manual Approval

- It must be possible for the user to approve the right one in case of multiple possible entries found
- It must be possible for the user to also conduct a search on his own and not to accept any of the proposals, if none fits

## Manual Search

- It must be possible for the user to conduct a manual search for a video file and to attach the metadata based on his search terms.
- This search must also be possible if we ask for manual approval, but none fits

## Updates

- It is possible to change the metadata association
- It is even possible to change a movie to a series and vice versa

# Future features

Additional feature which will have an impact on this feature will be implemented later. They are not yet to consider, but they might provide hints in terms of design and architecture so that they will be more easily implementable.

- none so far

# Tasks

- Analyze the requirements
- Check them for contradictions
- Check them for completeness
- Provide a design how you would store those data for movies and for series
- Create a comprehensive design how this feature will be implemented
- Break the feature down in reasonable tasks to conduct
- Implement the feature including tests
- Always ask missing information

# Constrains

- Keep the architecture, design and coding style of our current project if you use code from the old server
- Always ask if you see improvements compared to the old code
- This is a development project. We don't have running servers. So all breaking changes are fine and impose no issues.
