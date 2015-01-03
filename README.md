## Contributing Articles

If you'd like to contribute an article, please fork this repository, add your
article and create a pull request. Your articles should go in the `content/`
directory and your post images should go in `public/postimages/`. Please notice
that the article metadata needs to be at the very top in between the `+++`,
like so:

```
+++
author = ["Miles Davis"]
date = "1959-11-25T00:00:00-08:00"
title = "So What"
series = ["Birthday Bash 2014"]
draft = true
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

## Viewing the blog locally

To view the site on your local machine, you need to do the following:

1. Clone the repo
2. Install [Hugo](http://hugo.spf13.com)

Once Hugo is installed, run it from the cloned repo using:

	hugo server --watch

To view the site, visit the link provided by Hugo, usually `http://localhost:1313`.
