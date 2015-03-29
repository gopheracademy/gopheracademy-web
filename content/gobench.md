+++
author = ["Keiji Yoshida"]
date = "2015-03-27T15:07:34+09:00"
linktitle = "gobench - Go Benchmark Competition"
title = "gobench - Go Benchmark Competition"
+++

## Introduction

Simplicity is one of the philosophies of Go. Rob Pike said that the secret of Go's success is in its simplicity at [Go Conference 2014 autumn](http://gocon.connpass.com/event/9748/) in Tokyo, Japan. Dave Cheney [emphasized the importance of simplicity in Go](http://dave.cheney.net/2015/03/08/simplicity-and-collaboration) at [GopherCon India 2015](http://www.gophercon.in/). There are plenty of articles which explain the ways of writing code effectively in Go including [the official website of Go](http://golang.org/) and we can learn how to write simple Go code by studying these.

However, what about performance? How can we learn the ways of writing fast code in Go and improve the performance of our programs? The skill of writing fast code is as important as the one of writing simple code but the former is more difficult to master because much of the skill of writing fast code comes from experience.

So, I started [Go Benchmark Competition](http://gobench.org/), an environment in which we can gain practical experience of writing fast code in Go and exchange knowledge and know-how relating performance with one another. Now it's warmly welcomed by some of Go programmers and known affectionately as *gobench*.

![gobench](/postimages/gobench/gobench.png)

## Abstract

*gobench* competitions are held on [Slack](https://slack.com/). Participants in a competition can submit their code by simply sending it to a bot on Slack. The submitted code is automatically benchmarked and the bot on Slack notifies the submitter of its result. Rankings of competitions are shown in real time on [the official website](http://gobench.org/) like [this](http://gobench.org/results.html?no=1). Participants can improve and submit their code over and over again so that they can achieve higher rank. The submitted code is opened to the public after the competition ends. Participants can read and study from others' code and exchange their ideas and know-how with one another.

![slack](/postimages/gobench/slack.png)

## Merit of a Competition

A competition is the best thing in that it can give us an incentive to write faster code. A ranking page like [this](http://gobench.org/results.html?no=1) shows every submitted code's performance clearly and that prompts us to write faster code to achieve higher rank. So, a competition accentuates the difference between one's skill of writing fast code with others' and urges us to improve and master the skill.

![gobench](/postimages/gobench/competition.png)

## How to Join

You can join a competition easily by simply interacting with a bot on Slack. You can see how to join a competition in detail on the [How to Join](https://github.com/gobench/competitions/wiki/How-to-Join) Wiki page.

## Conclusion

I believe *gobench* will be useful to other Go programmers because it encourages them to improve the skill of writing fast code through collaborative learning from others. Or it might offer a good opportunity for beginners of Go to understand the importance of writing code considering its performance. I hope the skills/know-how you have improved/acquired at *gobench* will help you when you try to solve actual problems in Go.

If you have any questions or feedback about *gobench*, feel free to contact [me](https://twitter.com/_yosssi).
