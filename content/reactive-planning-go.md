+++
author = ["Gianluca Arbezzano"]
date = "2019-11-19T15:07:34+09:00"
linktitle = "Reactive planning and reconciliation in Go"
title = "Reactive planning and reconciliation in Go"
+++

I wrote a quick introduction about why I think [reactive planning is a cloud
native pattern](/blog/reactive-planning-is-a-cloud-native-pattern) and I
published an article about [control theory](/blog/control-theory-is-dope), but I
have just scratched the surface of this topic obviously. I have a 470 pages to
read from the book [Designing Distributed Control Systems: A Pattern Language
Approach](https://www.amazon.it/Designing-Distributed-Control-Systems-Veli-Pekka/dp/B01FIX9LMG).
It will take me forever.

## Introduction

It is easier to explain how much powerful reactive planning is looking at one
example, I wrote it in go, and in this article I am explaining the most
important parts.

Just to summarize, I think resiliency in modern application is crucial and very
hard to achieve in practice, mainly because we need to implement and learn a set
of patterns and rules. When I think about a solid application inside a
microservices environment, or in a high distributed ecosystem my mind drives me
into a different industry. I think about tractors, boilers, and what ever
does not depend on a static state stored inside a database but on a dynamic
source of truth.

When I think about an orchestrator it is clear to me that there is no way to
trust a cache layer in order to understand how many resources (VMs, containers,
pods) are running. We need to check them live because you never know what is
happening to your workload. Those kinds of applications are sensible to latency,
and they require a fast feedback loop.

That's one of the reason about why when you read about Kubernetes internals you
read about reconciliation loops and informers.

## Our use case

I wrote a small PoC, it is an application that I called
[cucumber](https://github.com/gianarb/cucumber), it is available on GitHub and
you can run it if you like.

It is a CLI tools that provisions a set of resources on AWS. The provisioned
architecture is very simple. You can define a number of EC2 and, they will be
created and assigned to a Route53 record, when the record does not exist the
application will create it.

I learned about how to think about problems like that. At the beginning of my
career the approach was simple, "I know what to do, I need to write a program
that reads the request and does what need to be done". So you start configuring
the AWS client, parsing the request and making a few API requests.

Everything runs perfectly and you succeed at creating 100 clusters.
Thing starts to be more complicated, you have more resources to provisioning
like load balancers, subnets, security groups and more business logic related to
who can do what. Requests start to be more than 5 at execution and the logic
somethings does not work as linear as it was doing before. At this point you
have a lot of conditions and figuring out where the procedure failed and how to
fix the issue becomes very hard.

This is why my current approach is different when I recognize this kind of
pattern I always start from the current state of the system.

You can question the fact that at the first execution it is obvious that nothing
is there, you can just create what ever needs to be created. And I agree on
that, but assuming that you do not know your starting points drives you to implement
the workflow in a way that is idempotent. When you achieve this goal you can
re-run the same workflow over and over again, if there is nothing to do the
program won't do anything otherwise it is smart enough to realize what needs to
be done. In this way you can create something called **reconciliation loop**.

## Reconciliation loop

The idea to re-run the procedures over and over assuming you do not know where
you left it is very powerful. Following our example, if the creation flow does
not end because AWS returned a 500 you won't be stuck in a situation where you
do not know how to end the procedure, you will just wait for the next
re-execution of the flow and it will be able to figure what is already created.
In my example this patterns works great when provisioning the route53 DNS record
because the DNS propagation can take a lot of time and in order to realize if
the DNS record already exists or if there are the right amounts of IPs attached
to it I use the
[`net.LookupIP`](https://jameshfisher.com/2017/08/03/golang-dns-lookup/), it
is the perfect procedure that can take an unknown amount of time to be
addressed.

## Reactive planning

At the very least reconciliation loop can be explained as a simple `loop` that
will execute a procedure forever but how do you write a workflow that is able to
understand the state of the system and autonomously make a plan to fix the gap
between current and desired state? This is what reactive planning does and
that's why control theory is done!

```go
// Procedure describe every single step to be executed. It is the smallest unit
// of work in a plan.
type Procedure interface {
	// Name identifies a specific procedure.
	Name() string
	// Do execute the business logic for a specific procedure.
	Do(ctx context.Context) ([]Procedure, error)
}

// Plan describe a set of procedures and the way to calculate them
type Plan interface {
	// Create returns the list of procedures that needs to be executed.
	Create(ctx context.Context) ([]Procedure, error)
	// Name identifies a specific plan
	Name() string
}
```

Let's start with a bit of Go. `Procedure` and `Plan` are the fundamental
interfaces to get familiar with:

* `Plan` are a collection of `Procedures`. The `Create` function is able to
  figure out the state of system adding procedures dynamically
* `Procedure` are the unit of work, they need to be as small as possible. The
  cool part about them is that they can return other procedures (and they can
  return other procedures as well) in this way build resilience. If a procedure
  returns an error the `Plan` is marked as failed.

```go
// Scheduler takes a plan and it executes it.
type Scheduler struct {
	// stepCounter keep track of the number of steps exectued by the scheduler.
	// It is used for debug and logged out at the end of every execution.
	stepCounter int
	// logger is an instance of the zap.Logger
	logger *zap.Logger
}

```

`Plan` and `Procedure` are crucial, but we need a way to execute a plan, it is
called scheduler. The `Scheduler` has an `Execture` function that accept a
`Plan` and it executes it **until there is nothing left to do**. Procedures can
returns other procedures it means that the scheduler needs to recursively
execute all the procedures.

The way the scheduler has to understand where the plan is done if via the number
of steps returned by the `Plan.Create` function. The scheduler executes every
plan at last twice, if the second time there are not steps left it means that
the first execution succeeded.

```go
// Execute accept an plan as input and it execute it until there are not anymore
// procedures to do
func (s *Scheduler) Execute(ctx context.Context, p Plan) error {
	uuidGenerator := uuid.New()
	logger := s.logger.With(zap.String("execution_id", uuidGenerator.String()))
	start := time.Now()
	if loggableP, ok := p.(Loggable); ok {
		loggableP.WithLogger(logger)
	}
	logger.Info("Started execution plan " + p.Name())
	s.stepCounter = 0
	for {
		steps, err := p.Create(ctx)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		if len(steps) == 0 {
			break
		}
		err = s.react(ctx, steps, logger)
		if err != nil {
			logger.Error(err.Error(), zap.String("execution_time", time.Since(start).String()), zap.Int("step_executed", s.stepCounter))
			return err
		}
	}
	logger.Info("Plan executed without errors.", zap.String("execution_time", time.Since(start).String()), zap.Int("step_executed", s.stepCounter))
	return nil
}
```

The `react` function implements the recursion and as you can see is the place
where the procedures get executed `step.Do`.

```go
// react is a recursive function that goes over all the steps and the one
// returned by previous steps until the plan does not return anymore steps
func (s *Scheduler) react(ctx context.Context, steps []Procedure, logger *zap.Logger) error {
	for _, step := range steps {
		s.stepCounter = s.stepCounter + 1
		if loggableS, ok := step.(Loggable); ok {
			loggableS.WithLogger(logger)
		}
		innerSteps, err := step.Do(ctx)
		if err != nil {
			logger.Error("Step failed.", zap.String("step", step.Name()), zap.Error(err))
			return err
		}
		if len(innerSteps) > 0 {
			if err := s.react(ctx, innerSteps, logger); err != nil {
				return err
			}
		}
	}
	return nil
}
```

All the primitives described in this section are in their go module called
[github.com/gianarb/planner](https://github.com/gianarb/planner) that you can
use. Other than what showed here the scheduler supports context cancellation and
deadline. In this way you can set a timeout for every execution.

One of the next big feature I will develop is a reusable reconciliation loop for
plans. In cucumber, it is very raw. Just a goroutine and a WaitGroup to keep the main
process up:

```
go func() {
    logger := logger.With(zap.String("from", "reconciliation"))
    scheduler.WithLogger(logger)
    for {
        logger.Info("reconciliation loop started")
        if err := scheduler.Execute(ctx, &p); err != nil {
            logger.With(zap.Error(err)).Warn("cucumber reconciliation failed.")
        }
        time.Sleep(10 * time.Second)
        logger.Info("reconciliation loop ended")
    }
}()
```
But this is too simple and it does not work in a distributed environment where
only one process should run the reconciliation and not all the replicas.

I wrote this code to help myself to internalize and explain what reactive
plans means. And also because I think the go community has a lot of great tools
that make uses of this concept like Terraform, Kubernetes but there are not low
level or simple to understand pieces of code. The next chapter describes how to
write your own control plan using reactive planning.

## Theory applied to cucumber...

Let's start looking at the `main` function:

```go
p := plan.CreatePlan{
    ClusterName:  req.Name,
    NodesNumber:  req.NodesNumber,
    DNSRecord:    req.DNSName,
    HostedZoneID: hostedZoneID,
    Tags: map[string]string{
        "app":          "cucumber",
        "cluster-name": req.Name,
    },
}

scheduler := planner.NewScheduler()
scheduler.WithLogger(logger)

if err := scheduler.Execute(ctx, &p); err != nil {
    logger.With(zap.Error(err)).Fatal("cucumber ended with an error")
}
```

In cucumber there is only one Plan the `CreationPlan`. We create it based on the
YAML file that contains the requested cluster. For example:

```yaml
name: yuppie
nodes_num: 3
dns_name: yeppie.pluto.net
```

And it gets executed by the scheduler. As you can see if the schedule returns an
error we do not exit, we do not kill the process. We do not panic! Because we
know that things can break and our code is designed to break just a little
and in way it can be recovered.

After the first execution the process spins up a goroutine that is the one I
copied above to explain a raw and simple control loop.

The process stays in the loop until we kill the process.

In order to test the reconciliation you can try to remove one or more EC2 or the
DNS record, watching the logs you will see how inside the loop the scheduler
executes the plan and reconcile the state of the system in AWS with the one you
described in the YAML.

```bash
CUCUMBER_MODE=reconcile AWS_HOSTED_ZONE=<hosted-zone-id> AWS_PROFILE=credentials CUCUMBER_REQUEST=./test.yaml go run cmd/main.go 
```

This is the command I uses to start the process.

The steps I wrote in cucumber are not many and you can find them inside
`./cucumber/step`:

1. create_dns_record
2. reconcile_nodes
3. run_instance
4. update_dns_record

`run_instance` for example is very small, it interacts with AWS via the go-sdk
and it creates an EC2:

```go
package step

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gianarb/planner"
	"go.uber.org/zap"
)

type RunInstance struct {
	EC2svc   *ec2.EC2
	Tags     map[string]string
	VpcID    *string
	SubnetID *string
	logger   *zap.Logger
}

func (s *RunInstance) Name() string {
	return "run-instance"
}

func (s *RunInstance) Do(ctx context.Context) ([]planner.Procedure, error) {
	tags := []*ec2.Tag{}
	for k, v := range s.Tags {
		if k == "cluster-name" {
			tags = append(tags, &ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String(v),
			})
		}
		tags = append(tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	steps := []planner.Procedure{}
	instanceInput := &ec2.RunInstancesInput{
		ImageId:      aws.String("ami-0378588b4ae11ec24"),
		InstanceType: aws.String("t2.micro"),
		//UserData:              &userData,
		MinCount: aws.Int64(1),
		MaxCount: aws.Int64(1),
		SubnetId: s.SubnetID,
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags:         tags,
			},
		},
	}
	_, err := s.EC2svc.RunInstances(instanceInput)
	if err != nil {
		return steps, err
	}
	return steps, nil
}
```

As you can see the unique situation where I return an error is if the
`ec2.RunInstance` fails, but only because this is a simple implementation.
Moving forward you can replace the return of that error with other steps, for
example you can terminate the cluster and cleanup, in this way you won't left
broken cluster around, or if you try other steps to recover from that error
leaving at the next executions (from the reconciliation loop) to end the
workflow.

From my experience reactive planning makes refactoring or development very
modular, as you can see you do not need to make all the flow rock solid since
day one, because it is very time-consuming, but you always have a clear
entrypoint for future work. Everywhere you return or log an error can be
replaced at some point with steps, making your flow rock solid from the
observation you do from previous run.

The `reconcile_nodes` is another interesting steps. Because `run_insance` only
calls AWS and it creates one node, but as you can image we need to create or
terminate a random amount of them depending on the current state of the system.

1. if you required 3 EC2 but there are zero of them you need to run 3 new nodes
2. if there are 2 nodes but your required 3 we need 1 more
3. if there are 56 nodes but you required 3 of them we need to terminate 63 EC2s

The `reconcile_nodes` procedures makes that calculation and returns the right
steps:

```go
package step

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/zap"

	"github.com/gianarb/planner"
)

type ReconcileNodes struct {
	EC2svc        *ec2.EC2
	Tags          map[string]string
	VpcID         *string
	SubnetID      *string
	CurrentNumber int
	DesiredNumber int
	logger        *zap.Logger
}

func (s *ReconcileNodes) Name() string {
	return "reconcile-node"
}

func (s *ReconcileNodes) Do(ctx context.Context) ([]planner.Procedure, error) {
	s.logger.Info("need to reconcile number of running nodes", zap.Int("current", s.CurrentNumber), zap.Int("desired", s.DesiredNumber))
	steps := []planner.Procedure{}
	if s.CurrentNumber > s.DesiredNumber {
		for ii := s.DesiredNumber; ii < s.CurrentNumber; ii++ {
			// TODO: remove instances if they are too many
		}
	} else {
		ii := s.CurrentNumber
		if ii == 0 {
			ii = ii + 1
		}
		for i := ii; i < s.DesiredNumber; i++ {
			steps = append(steps, &RunInstance{
				EC2svc:   s.EC2svc,
				Tags:     s.Tags,
				VpcID:    s.VpcID,
				SubnetID: s.SubnetID,
			})
		}
	}
	return steps, nil
}
```

As you can see I have only implemented the `RunInstnace` step, and there is a
`TODO` left in the code, it means that scale down does not work for now.
It returns the right steps required to matches the desired state, if there are 2
nodes, but we required 3 of them this steps will return only one `RunInstance`
that will be executed by the scheduler.

Last interesting part of the code is the `CreatePlan.Create` function. This is
where the magic happens. As we saw the schedulers calls the `Create` functions
at least twice for every execution and its responsability is to measure the
current state and according to it calculate the required steps to achieve that
we desire. It is a long function that you have in the repo, but this is an idea:

```go
resp, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
    Filters: []*ec2.Filter{
        {
            Name:   aws.String("instance-state-name"),
            Values: []*string{aws.String("pending"), aws.String("running")},
        },
        {
            Name:   aws.String("tag:cluster-name"),
            Values: []*string{aws.String(p.ClusterName)},
        },
        {
            Name:   aws.String("tag:app"),
            Values: []*string{aws.String("cucumber")},
        },
    },
})

if err != nil {
    return nil, err
}

currentInstances := countInstancesByResp(resp)
if len(currentInstances) != p.NodesNumber {
    steps = append(steps, &step.ReconcileNodes{
        EC2svc:        ec2Svc,
        Tags:          p.Tags,
        VpcID:         vpcID,
        SubnetID:      subnetID,
        CurrentNumber: len(currentInstances),
        DesiredNumber: p.NodesNumber,
    })
}
```
The code checks if the number of running instances are equals to the desired
one, if they are different it calls the `ReconcileNodes` procedure.

## Conclusion

This is it! It is a long article but there is code and a repository you can run!
I am enthusiast about this pattern and the work exposed here because I think it
makes it clear and I tried to keep the context as small as possible to stay
focused on the workflow and the design.

Let me know if you will end up using it! Or if you already do how it is going
[@gianarb](https://twitter.com/gianarb).
