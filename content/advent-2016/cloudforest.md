+++
title = "Predicting genetic diseases with CloudForest"
author = ["Vitor De Mario"]
date = "2016-12-23T00:00:00"
series = ["Advent 2016"]
+++

[CloudForest](https://github.com/ryanbressler/CloudForest) is a machine learning project dedicated to the construction of [Random Forests](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm) built entirely in Go. It was created by [Ryan Bressler](https://github.com/ryanbressler).

Random Forests are a machine learning algorithm based around the construction of many single classification trees, each splitting both the training set and the features available to train the model randomly. Each single tree is different from the others due to this random split and the ensemble of all the trees together is able to classify the data better than any single tree could do by itself. We won't go deep on how Random Forests work internally, you can learn more [here](https://www.stat.berkeley.edu/~breiman/RandomForests/cc_home.htm#intro).

CloudForest is a swiss army knife of options for how a model can be built. It supports not only the original ideas from Breiman and Cutler's algorithm but also boosting, class weighted classification, feature selection with [artificial constrasts](http://www.jmlr.org/papers/volume10/tuv09a/tuv09a.pdf), support for missing values and many more features.

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

CloudForest was built as a Go package. We've seen only command line utilities so far, but these command line utilities are just `main` packages that import the base package and use its functions. Any other Go program can do the same.

For example, loading an AFM file into memory can be done like this:


```
import (
	"os"
	"github.com/ryanbressler/CloudForest
)

func main() {
	file, err := os.Open("file.afm")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	matrix, err := CloudForest.LoadAFM(*fm)
	if err != nil {
		log.Fatal(err)
	}
	(...)
}
```

Loading a forest:


```
forestFile, err := os.Open("forest.sf")
if err != nil {
	log.Fatal(err)
}
defer forestFile.Close()
forestReader := CloudForest.NewForestReader(forestFile)
forest, err := forestreader.ReadForest()
if err != nil {
	log.Fatal(err)
}
```

To apply a forest onto a set of data a `VoteTallyer` is necessary. There are a few `VoteTallyer` implementations ready to use in the CloudForest package, such as `CatBallotBox`, which is appropriate for categorical classification and `NumBallotBox`, which is useful to average predictions for numerical features.

Combining the data matrix from our first snippet and the forest from our second, we could apply the forest over all the cases in the matrix, tallying the results as a boolean prediction:

```
var ballotBox CloudForest.VoteTallyer
ballotBox = CloudForest.NewCatBallotBox(matrix.Data[0].Length())

for _, tree := range forest.Trees {
	tree.Vote(matrix, ballotBox)
}

for i := range matrix.CaseLabels {
	tally := ballotBox.Tally(i)
	prediction, _ := strconv.ParseBool(tally)
}
```

Binary (boolean) classification is only one possibility. With different vote tallyers, different target features and options, several other kinds of classifications can be done.

# Real-world use case: predicting mutations that cause diseases

At [Mendelics](https://github.com/mendelics) in Brazil, we've been using CloudForest for almost 3 years in the field of genetics (spoilers: the last code snippet was from a real system). We've started using CloudForest almost at the same time as we've started using Go, with impressive results from both.

The human genome is over 3 billion nucleotides long. Every person has thousands of mutations (the term _variant_ is technicaly more correct, but we'll keep using _mutation_), defining their distinct features, the characteristics that make them unique. Unfortunately, some of these mutations, instead of changing the color of your eyes, cause diseases.

Determining which mutations cause diseases in the middle of this sea of mutations is not straightforward. Not every mutation is covered by scientific publications, and many disease causing mutations have never been seen even in huge databases of human mutations such as [ExAC](http://exac.broadinstitute.org/) or [gnomAD](http://gnomad.broadinstitute.org/) (thanks, natural selection). To solve this problem, researchers in genetics have [often](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC3154091/) turned to machine learning models built with Random Forests.

## Turning mutations into AFM rows

A mutation can cross many genes and [exons](https://en.wikipedia.org/wiki/Exon). Some of its features can be determined for the entire mutation and others depend on which _segment_ we're working with. Therefore, it makes sense to model a mutation as a struct with several slices and a somewhat deep hierarchy.


```
type Mutation struct {
	Chromosome  string
	Start       int
	End         int
	Reference   string
	Alternative string
	Annotations []Segment
}

type Segment struct {
	GeneName        string
	ExonNumber      string
	MaxConservation float64
	AminoacidsTotal int
	(...)
}
```

This deep hierarchy does not translate well into a simple tsv row. Another problem is how to select which features we want to use. In this example, we probably don't want to use the `Chromosome` field in the classification model, the structural characteristics of the mutation are the real driving factors to whether it causes a disease or not, so we might as well completely ignore it when dealing with CloudForest.

To solve both of these issues, we turned to reflection and struct tags.

```
type Mutation struct {
	Chromosome  string
	Start       int
	End         int
	Reference   string
	Alternative string
	Annotations []Segment `AFM:"true"`
}

type Segment struct {
	GeneName        string
	ExonNumber      string
	MaxConservation float64 `AFM:"true"`
	AminoacidsTotal int     `AFM:"true"`
	(...)
}
```

With reflection, we traverse `Mutation`s in memory, creating tsv rows for each `Segment` and repeating the common values shared by each segment. The `AFM` struct tag determines if a feature is used or discarded. The end result is a complete afm file built in memory during execution of the program.

```
// afmTags checks if a struct field has the AFM tag
func afmTags(v reflect.StructField) bool {
	return v.Tag.Get("AFM") != ""
}

// afmPrefix translates Go types into the prefixes CloudForest expects on the AFM header
func afmPrefix(k reflect.Kind) string {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return "N"
	case reflect.Bool:
		return "B"
	case reflect.String:
		return "C"
	}
	return ""
}

// processHeader iterates over all of the Mutation fields translating the field into the format expected
// for the AFM header. Processing a segment is equivalent, using the struct field value instead of its name.
func processHeader(mutation Mutation) []string {
	reflection := reflect.ValueOf(mutation)
	reflectionType := reflection.Type()
	fields := make([]string, 0, reflection.NumField())
	for i := 0; i < reflection.NumField(); i++ {
		structValue := reflection.Field(i)
		structField := reflectionType.Field(i)
		isAFM := afmTags(structField)
		if isAFM {
			// omitting treatment for slices and nested structs
			prefix := afmPrefix(structValue.Kind())
			name := strings.ToLower(structField.Name)
			field := fmt.Sprintf("%s:%s", prefix, name)
			fields = append(fields, field)
		}
	}
	return fields
}
```

Forest construction is still done in the command line with `growforest`. Application of the model to new, unknown mutations, is done entirely in memory, as the system receives new files for processing. At the time this was originally built, without much experience in Go, we did not realize that all of this work was actually the construction of an encoding library. It does not conform to the encoding interfaces in the standard library due to that oversight and has never seen the light of open source, sadly.

In production, we've had multiple models built with `growforest` and applied this way, always achieving more than 90% accuracy accross all of our tests. The system can't predict by itself every single mutation that causes a disease in real patients but serves as a powerful tool for the geneticists to quickly find what really matters.

# Conclusion

Random Forests are a general purpose algorithm suited for many applications and CloudForest has been around for many years, proving itself one of the most robust Machine Learning libraries built with Go so far.

Try it for yourself and get in touch with the community at [the Google Group](https://groups.google.com/forum/#!forum/cloudforest-dev)!

Bonus points: if you can understand Portuguese, the story of the construction of the genetics system was the talk that opened the second day of GopherCon Brazil's first edition. The video is up at https://youtu.be/KkQpT7acNFc.
