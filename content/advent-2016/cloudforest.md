+++
title = "Predicting genetic diseases with CloudForest"
author = ["Vitor De Mario"]
date = "2016-12-23T00:00:00"
series = ["Advent 2016"]
+++

[CloudForest](https://github.com/ryanbressler/CloudForest) is a machine learning project dedicated to the construction of [Random Forests](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm) built entirely in Go. It was created by [Ryan Bressler](https://github.com/ryanbressler).

Random Forests are a machine learning algorithm based around the construction of many single classification trees, each splitting both the training set and the features available to train the model randomly. Each single tree is different from the others due to this random split and the ensemble of all the trees together is able to classify the data better than any single tree could do by itself. We won't go deep on how Random Forests work internally, you can learn more [here](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm#intro).

# CloudForest

CloudForest is a swiss army knife of options for how a model can be built. It supports not only the original ideas from Breiman and Cutler's but also boosting, class weighted classification, feature selection with artificial constrasts, support for missing values and many more features.

The list of flags for all these options can feel daunting when you're not familiar with the terms, but hang on, we'll explore most of it.

![CloudForest options](/postimages/advent-2016/cloudforest/controlpanel.png)

# Growing a forest

To start building a classification model, we're gonna need the `growforest` command, which you can get by running:

```
go get github.com/ryanbressler/CloudForest
go install github.com/ryanbressler/CloudForest/growforest
```

With `growforest` available, training your first model is as easy as running:

`growforest -train train.fm -rfpred forest.sf -target B:FeatureName`

These three flags are the bread and butter of `growforest`. `-train` specifies an input file, containing all the training data CloudForest will use to build the model, `-rfpred` is the name of the output file where the forest will be written to and `-target` is the name of the feature we are trying to predict.

With these options, `growforest` will build a forest with 100 trees, using every feature available in the training set to correlate with the target. The `.sf` file generated is just a list of trees, each line representing a decision point based on one of the features.

# Applying a forest

After we build our model with `growforest`, we'll want to use it to analyze new data. For that, we'll need the `applyforest` command.

```
go install github.com/ryanbressler/CloudForest/applyforest
```

Applying a model built with `growforest` to new data is also simple:

`applyforest -fm test.fm -rfpred forest.sf`

The `-fm` flag specifies the filename with the test set data, akin to the `-train` flag we've used for `growforest`.

# The AFM format

# Feature importance

# Extra options for growforest

maxDepth
mTry
nTrees
blacklist
progress
test

# Importing and using CloudForest directly from other Go programs

# Real-world use case: predicting mutations that cause diseases

# Conclusion

community, google group
