## Contributing Articles

If you'd like to contribute an article, please fork this repository, add your
article and create a pull request. Your articles should go in the `upcoming`
directory and your post images should go in `public/postimages/`. Please notice
that the article metadata needs to be at the very top in between thee `+++`,
like so:

```
+++
author = ["Miles Davis"]
date = "1959-11-25T00:00:00-08:00"
title = "So What"
series = ["Birthday Bash 2014"]
+++
```

The easiest way to do this is to have [hugo](http://gohugo.io) create
the new post for you.

    hugo new "section/title of post"

For example if I was writing a post for the 2014 advent called "go awesome":

    hugo new "advent-2014/go-awesome.md"

Hugo will automatically create the file and put the proper metadata in place.
Just make sure to review the metadata and adjust as needed.

## Style Guide

Blog posts should be formatted using appropriate markdown. Please make
sure to properly wrap lines for maximum readability, 72 columns is a
good standard to apply. Please ensure no spelling or typographical
errors are present. Make sure to preview your content locally to ensure
that it looks correct before submitting a pull request.

## Working with Hugo

There are two parts to the code now. The main Hugo app still runs the blog with
**config.toml** while an additional **config-main.toml** runs the main website.

1. Clone the repo
2. Install [Hugo](http://hugo.spf13.com)

## Working with the blog 

Theme/HTML are in `layouts/` and static assets(CSS/JS) are in `static/`. When
Hugo runs, the final layout is generated and served from the `public/` folder.
Run the Hugo server with:

	hugo server --watch

## Working with the main site

The website runs on the same Hugo app but has a few things configured
differently. Layouts are in `layouts-main/` and generated pages are in
`public-main/`. It uses the same `content/` folder from the blog app to pull
info to the main site. Please see `config-main.toml` to understand how it's
configured. 

To run the server, include the mentioned config file as a flag:

In the gopheracademy-web directory:

    hugo server --watch --config="config-main.toml"


To view the site, visit the link provided by Hugo, usually `http://localhost:1313`.
