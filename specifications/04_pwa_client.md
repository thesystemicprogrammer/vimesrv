# Specifications for the Feature PWA Client

I want to build a media sever that serves video in our home network. This describe the feature on how to structure the already existing minimal pwa player.

# Basic idea

You are a versed user experience designer and you know how to build compelling video server clients. It should be easy and fun to use them. You do a lot of research and learn from the best in the field how to setup things properly.

# Requirements

## Auth

- As a user, I want to be able to login with my user ID and password
- As a user, I cannot sign in. I need an admin that has prepared my account with an initial password
- As a user, if I forgot my password, I can reset it
- As an admin, I can create new users
- As an admin, I can assign a role to a user: admin, manager, or user
- As an admin, I can reset passwords of existing users
- As an admin, I can change roles of users
- As an admin, I can delete users

## Library

- As a user, I get a nice overview over my content. You as an UI expert know exactly how to set this up nicely
- I see my movies and my series nicely ordered with posters from TMDB
- When I click a movie I will be forwarded to the movie details
- When I click a series, I will be forwarded to the series details

## Movie Details

- Here I see the details of a movie with long description
- Here I can go back to the library
- Here I can start the player to show me the selected movie

## Series Details

- Here I see the details for my series. I see the description of the series
- Below I see the seasons listed with the description per season
- Below I see all the episodes of a episode listed with a description
- Here I can start the player to show me the episode

## Metadata

- As a user, I can here see which of my media files need an approval as there are multiple fitting metadata entry and it could not be automatically determined which is the right one
- As a user I can select the right metadata entry for media files waiting for approval or I can start a new search if none is fitting
- As a user I can conduct a manual search for media files that have not yet found metadata or are waiting for approval but no entry is fitting
- As a user I can change the metadata for all my files if I want to change it

## Player

- The player allows to select the audio track
- The player allows to select the subtitle
- The player plays the actual media

# Future features

Additional feature which will have an impact on this feature will be implemented later. They are not yet to consider, but they might provide hints in terms of design and architecture so that they will be more easily implementable.

- none so far

# Tasks

- Analyze the requirements
- Check them for contradictions
- Check them for completeness
- Create a comprehensive design how this feature will be implemented
- Analzye the additionally required endpoints and adaptions in the server part
- Break the feature down in reasonable tasks to conduct
- Implement the feature including tests
- Always ask missing information

# Constrains

- Keep the architecture, design and coding style of our current project if you use code from the old server
- Always ask if you see improvements compared to the old code
- This is a development project. We don't have running servers. So all breaking changes are fine and impose no issues.
