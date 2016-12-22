+++
title = "Predicting genetic diseases with CloudForest"
author = ["Vitor De Mario"]
date = "2016-12-23T00:00:00"
series = ["Advent 2016"]
+++

[CloudForest](https://github.com/ryanbressler/CloudForest) is a machine learning project dedicated to the construction of [Random Forests](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm) built entirely in Go. It was created by [Ryan Bressler](https://github.com/ryanbressler).

Random Forests are a machine learning algorithm based around the construction of many single classification trees, each splitting both the training set and the features available to train the model randomly. Each single tree is different from the others due to this random split and the ensemble of all the trees together is able to classify the data better than any single tree could do by itself. We won't go deep on how Random Forests work internally, you can learn more [here](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm#intro).

CloudForest is a swiss army knife of options for how a model can be built. It supports not only the original ideas from Breiman and Cutler's algorithm but also boosting, class weighted classification, feature selection with artificial constrasts, support for missing values and many more features.

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

We've talked about the input files for the training set and the test set but we haven't discussed in which format these files are expected. CloudForest expects input files to be in Annotated Feature Matrix (.afm) format. It also supports .libsvm and .arff formats, but the standard is .afm.

AFM, as the name implies, is a matrix. The file is a tsv, a sequence of columns separated by tabs. In this tsv, either the first row or the first column is a set of headers describing each feature present in the dataset. By default the columns represent entries in the dataset and the rows represent features. The opposite is also supported, and is often easier to work with.

Each row/column header includes a prefix to the name of the feature. The presence of these prefixes is what determines if the file is oriented by column or row. There are three prefixes:

- N: Numerical feature.
- C: Categorical feature.
- B: Boolean feature.

Every feature has to be mapped to one of these prefixes. Numerical features can be both integers and floating point numbers. Categorical and boolean features are represented by strings.

# Feature importance

One of the most interesting aspects of building classification models with CloudForest is finding out which features have the biggest impact on your model. Not every feature impacts the model in the same way, some are highly correlated with what you're trying to predict and some not so much. Knowing which features are important can help you iterate and build better models.

When you're running `growforest`, if you specify the flag `-importance=filename.tsv`, the `growforest` command will create a second output file along with the `.sf` file. This new file is a tsv file that lists every feature along with several metrics.

Some of these metrics deal with how many trees chose to use a feature, others show how a feature impacted the accuracy of the whole model. The full list of metrics can be found [here](https://github.com/ryanbressler/CloudForest#importance).

# Extra options for growforest

To support all of the different ways CloudForest can build classification models, there are dozens of flags. The list of flags for all the options can feel daunting when you're not familiar with some of the terms, much like this:

![CloudForest options](/postimages/advent-2016/cloudforest/controlpanel.png)

However, some of the flags are of particular importance to any kind of model:

- `nTrees`: the number of trees. By default, if you don't specify anything for this option, the forest will be built with `-nTrees=100`. A hundred trees if often a good enough number, but there is no one number of trees that will be appropriate for every situation.
- `progress`: report the tree number and the running error. With `-progress`, it is possible to follow the construction of the forest. The first trees usually have high errors but the model quickly adjusts itself towards smaller and smaller errors as more trees are built. `-progress` is a good companion to `-nTrees`, it can be used to find out empirically how many trees are a good fit for your model. If the forest construction converges early, there is little benefit to increasing the number of trees.
- `mTry`: the number of candidate features to be selected with each tree. Not every tree in a forest is build the same. Each classification tree is built with a subset of the features. `-mTry` determines how many features are available for selection during the construction of each tree. Lower values yield trees with less correlation and higher values allow more opportunities for important features to be selected. If not specified, `-mTry` is the square root of the number of features. This value is common across many implementations of Random Forests, and it's one of the parameters that has the highest impact on the accuracy of the final model. You may want to experiment with bigger values for `-mTry`. We've had success with much higher ratios, such as .5 (50% of the features available for each tree), but your mileage may vary.
- `blacklist`: a list of features that shouldn't be used by any of the trees. Sometimes, parts of the data on the .afm files aren't really features, only some kind of metadata that is not relevant for classification. This option excludes these features without requiring you to edit the .afm file used on `growforest`.
- `test`: a .afm file with more data used for testing. The data in this file is not used to generate the classification forest. After the model is complete, it is tested on this data and the accuracy results are reported in the end of `growforest`'s execution. This option is the equivalent of running `applyforest` right after finishing `growforest`.
- `maxDepth`: depending on your data, some trees may have a hard time converging. `-maxDepth` determines how tall a tree can grow before the algorithm cuts it. Without `-maxDepth`, `-growforest` can run fast at first, completing the construction of several trees and then slow down in the end when some of the feature sets have trouble finding a good way to split the cases.

# Importing and using CloudForest directly from other Go programs

# Real-world use case: predicting mutations that cause diseases

# Conclusion

community, google group
