+++
author = ["Ward K Harold"]
date = "2014-12-29T13:12:45-06:00"
title = "Cron as a file system"

+++

### 9p

I read [The Styx Architecture for Distributed
Systems](http://doc.cat-v.org/inferno/4th_edition/styx) over a decade ago.  The
central idea of the paper is that "representing a computing resource as a form
of file system, [makes] many of the difficulties of making that resource
available across the network disappear". By resource they mean *any* resource.
For example in the Plan9 window system 8½, windows and even the mouse are implemented
as a files; similarly, in Inferno and Plan9 the interface to the TCP/IP network is
presented as a file system hierarchy. The idea is elegant and practical and the
paper is a must read. Unfortunately, for a very long time the only way to
actually experiment with its ideas was to install Plan9 or Inferno.

Happily, things have changed in over the last few years. Several former Bell
Labs and/or Plan9 folk implemented 9p as a Linux kernel module, v9fs, and got
it integrated into the 3.x Linux kernels. As a result 9p is available out of
the box in modern Linux distros - at least it's there on Ubuntu 12.04 and later
and the latest Fedora releases. Having it in the kernel is fine but to actually
make use of it you need user space libraries. Intrepid developers have been
busy implementing the necessary
[libraries](http://9p.cat-v.org/implementations) in languages from C to
Haskell. My personal favorite is [go9p](https://code.google.com/p/go9p/). In
what follows I'll use it to implement a simple *cron* service as a 9p file
system.

### jobd design

When designing a system service or application using 9p we begin by designing a
suitable name space. Often there are two parts to such a name space, the static
part created when the application or service starts and the dynamic part that
gets filled in as users interact with the system. In our cron service the
static part of the name space will consist of a *clone* file that is used to
create new jobs and a *jobs* directory that individual jobs will live under:

```
.../clone
.../jobs
```

To create a new job the user opens the clone file and writes a job definition
string to it. A job definition has the form:

```
<job name>':'<cron expr>':'<command>
```
so a "hello world" job definition would be:

```
hello:0 0/1 * * * ? *:echo hello world
```

If we write that job definition to the *clone* file it will create a job named
'hello' that prints "hello world" every minute.  How is the new 'hello' job
represented in the name space? Like so:

```console
# assuming the jobd file system is mounted over /mnt/job.d
$ tree /mnt/job.d
/mnt/jobs.d
├── clone
└── jobs
    └── hello
        ├── cmd
        ├── ctl
        ├── log
        └── schedule
```

Jobs are the dynamic part of the name space. Each one is represented by a
directory corresponding to the name of the job. Under each job directory is a
collection of four files that allow users to control and monitor the job.

* writing the text string 'start', upper or lower case, to the ctl file, e.g.,
`echo -n start > ctl` starts the job and writing 'stop' to the ctl file stops the
job
* reading from the log file yields the last results of the last N executions of
the job, where N is configurable at job daemon start up
* reading the cmd file returns the command associated with the job, in case you
forgot
* finally, reading the schedule file returns the cron expression that
determines the execution schedule of the job and, if the job is started, the
next time it will execute

So that's the design of the name space of our cron service. I've also specified
the behavior of the system as the results of reading and writing the various
files in the name space. Note that none of the files or directories in the name
space actually exist on disk; they are synthetic files not unlike the files in
the /proc file system of a Linux system. The name space is presented to the
system by a daemon, jobd, that responds appropriately to the various messages
of the 9p protocol. The jobd daemon listens on a TCP/IP port. While
applications that understand 9p could connect directly to jobd it's much
simpler to just mount jobd and use the standard Linux file system interface to
interact with it.

We'll look at the actual code in a minute but for now assume that we've built
the jobd binary and started it up either from the command line or via a service
manager like systemd or supervisord. Then we can mount it via the mount
command:

```console
$ mount -t 9p -o protocol=tcp,port=5640 192.168.0.42 /mnt/jobs.d
```

and we can create and manage jobs using standard file system calls from any
programming language, or straight from the shell:

```console
$ ls /mnt/jobs.d
clone jobs
$ echo -n 'hello:0 0/1 * * * ? *:echo hello world' > /mnt/job.d/clone
$ ls /mnt/jobs.d/jobs
hello
$ cd /mnt/jobs.d/jobs/hello
$ echo -n start > ctl
```

### jobd implementation

The jobd source lives in the [wkharold/jobd](http://github.com/wkharold/jobd)
github repo. It's relatively small and, once you have a basic grasp of the go9p
machinery, fairly simple. One quick meta comment on the repo: there are a
couple of approaches to dealing with package dependencies in go, vendoring and
for lack of a more concise term what I'll call tool-based version control. I've
chosen the former because the packages jobd depends on:
[go9p](https://code.google.com/p/go9p), Rob Pike's
[glog](https://github.com/golang/glog), and Raymond Hill's
[cronexpr](https://github.com/gorhill/cronexpr), are stable and I find the
vendoring approach simpler to understand and manage - your milage may vary.

Jobd consists of three components:

1. the network server
1. the clone file that creates jobs
1. the per job collection of files that control and provide information about jobs

These components are all part of jobd's main package which is composed of four source files: jobd.go, clone.go, jobs.go, and job.go.

Let's look at the network server first. The points of interest here are the
creation of the static portion of the jobd name space and the firing up of the
network listener. The code below shows the `mkjobfs` function which creates the
jobd clone file and jobs directory.

```go
/*** jobd.go ***/
// mkjobfs creates the static portion of the jobd file hierarchy: the 'clone'
// file, and the 'jobs' directory at the root of the hierarchy.
func mkjobfs() (*srv.File, error) {
	var err error

	user := p.OsUsers.Uid2User(os.Geteuid())

	root := new(srv.File)

	err = root.Add(nil, "/", user, nil, p.DMDIR|0555, nil)
	if err != nil {
		return nil, err
	}

	err = mkCloneFile(root, user)
	if err != nil {
		return nil, err
	}

	jobsroot, err = mkJobsDir(root, user)
	if err != nil {
		return nil, err
	}

	return root, nil
}

/*** clone.go ***/
type clonefile struct {
    srv.File
}

// mkCloneFile creates the clone file at the root of the jobd name space.
func mkCloneFile(dir *srv.File, user p.User) error {
	glog.V(4).Infoln("Entering mkCloneFile(%v, %v)", dir, user)
	defer glog.V(4).Infoln("Exiting mkCloneFile(%v, %v)", dir, user)

	glog.V(3).Infoln("Create the clone file")

	k := new(clonefile)
	if err := k.Add(dir, "clone", user, nil, 0666, k); err != nil {
		glog.Errorln("Can't create clone file: ", err)
		return err
	}

	return nil
}

/*** jobs.go ***/
type jobsdir struct {
	srv.File
	user p.User
}

// mkJobsDir create the jobs directory at the root of the jobd name space.
func mkJobsDir(dir *srv.File, user p.User) (*jobsdir, error) {
	glog.V(4).Infof("Entering mkJobsDir(%v, %v)", dir, user)
	defer glog.V(4).Infof("Leaving mkJobsDir(%v, %v)", dir, user)

	glog.V(3).Infoln("Create the jobs directory")

	jobs := &jobsdir{user: user}
	if err := jobs.Add(dir, "jobs", user, nil, p.DMDIR|0555, jobs); err != nil {
		glog.Errorln("Can't create jobs directory ", err)
		return nil, err
	}

	return jobs, nil
}
```

The go9p srv package exposes a `File` type. As might be expected this is one of
the key components of a 9p based system. At the root of the jobd name space is
a directory named '/' which is created by the call to `root.Add` in `mkjobfs`.
Once the root is created the jobs directory is added via `mkJobsDir` where a
`jobsdir` struct is instantiated and then made a child of the jobd root, which
was passed in as `dir`. Similarly, `mkCloneFile` instantiates a `clonefile`
struct and makes it a child of the jobd root. Note, however, that since it's
just a regular file the `p.DMDIR` bits aren't a part of its permissions mask
(the 5th parameter of the `Add` invocation).

Starting up the network listener is pretty simple. In the code below the call
to `srv.NewFileSrv` instantiates a struct that holds all the fields necessary
for handling the 9p protocol. `s.Start` initializes it and starts up the
goroutines it uses, and finally after invoking `s.StartNewListener` jobd starts
listening for incoming connections.

```go
func main() {
    	// argument handling and initialization

	root, err := mkjobfs()
	if err != nil {
		os.Exit(1)
	}

    	// job database management

	s := srv.NewFileSrv(root)
	s.Dotu = true
	if *fldebug {
		s.Debuglevel = 1
	}
	s.Start(s)

	if err := s.StartNetListener("tcp", *flfsaddr); err != nil {
		glog.Errorf("listener failed to start (%v)", err)
		os.Exit(1)
	}

	os.Exit(0)
}
```

OK, on to the clone file. To add behavior to a 9p you implement one or more of
the standard file operations: Create, Open, Read, Write, Remove, etc. In the
case of the clone file the only supported operation is Write - writing a job
definition string to the clone file creates the corresponding job subtree in
the jobs directory. 

Here's the pertinent code.

```go
/*** clone.go ***/
type clonefile struct {
    srv.File
}

// Write handles writes to the clone file by attempting to parse the data being
// written into a job definition and if successful adding the corresponding job
// to the jobs directory.
func (k *clonefile) Write(fid *srv.FFid, data []byte, offset uint64) (int, error) {
	glog.V(4).Infof("Entering clonefile.Write(%v, %v, %v)", fid, data, offset)
	defer glog.V(4).Infof("Exiting clonefile.Write(%v, %v, %v)", fid, data, offset)

	k.Lock()
	defer k.Unlock()

	glog.V(3).Infof("Create a new job from: %s", string(data))

	jdparts := strings.Split(string(data), ":")
	if len(jdparts) != 3 {
		return 0, fmt.Errorf("invalid job definition: %s", string(data))
	}

	jd, err := mkJobDefinition(jdparts[0], jdparts[1], jdparts[2])
	if err != nil {
		return 0, err
	}

	if err := jobsroot.addJob(*jd); err != nil {
		return len(data), err
	}

	db, err := os.OpenFile(jobsdb, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return len(data), err
	}

	fmt.Fprintf(db, "%s\n", string(data))
	db.Close()

	return len(data), nil
}

/*** jobs.go ***/
type jobsdir struct {
	srv.File
	user p.User
}

// addJob uses mkJob to create a new job subtree for the given job definition and adds it to
// the jobd name space under the jobs directory.
func (jd *jobsdir) addJob(def jobdef) error {
	glog.V(4).Infof("Entering jobsdir.addJob(%s)", def)
	defer glog.V(4).Infof("Leaving jobsdir.addJob(%s)", def)

	glog.V(3).Info("Add job: ", def)

	job, err := mkJob(&jd.File, jd.user, def)
	if err != nil {
		return err
	}

	if err := job.Add(&jd.File, def.name, jd.user, nil, p.DMDIR|0555, job); err != nil {
		glog.Errorf("Can't add job %s to jobs directory", def.name)
		return err
	}

	return nil
}

/*** job.go ***/
type jobdef struct {
	name     string
	schedule string
	cmd      string
	state    string
}

// mkJobDefinition examines the components of a job definition it is given and
// returns a new jobdef struct containing them if they are valid.
func mkJobDefinition(name, schedule, cmd string) (*jobdef, error) {
	if ok, err := regexp.MatchString("[^[:word:]]", name); ok || err != nil {
		switch {
		case ok:
			return nil, fmt.Errorf("invalid job name: %s", name)
		default:
			return nil, err
		}
	}

	if _, err := cronexpr.Parse(schedule); err != nil {
		return nil, err
	}

	return &jobdef{name, schedule, cmd, STOPPED}, nil
}
```

A `Write` method is defined for the `clonefile` type. The `mkCloneFile` function
mentioned above instantiated a `clonefile` and made it a child of the jobd root.
When the jobd file system is mounted by the OS write operations will end up
invoking our `Write` method. It will receive the job definition string in the
`data` parameter as a slice of bytes. Those bytes get turned into a `jobdef`
which is then used to create the job subtree in the jobs directory.

Finally, let's look at the creation of the job subtree. A job is represented by
a directory containing four files: *ctl*, *cmd*, *log*, *schedule*. Here's
the code used to create those four files.

```go
/*** job.go ***/
type jobdef struct {
	name     string
	schedule string
	cmd      string
	state    string
}

type jobreader func() []byte
type jobwriter func([]byte) (int, error)

type job struct {
	srv.File
	defn    jobdef
	done    chan bool
	history *ring.Ring
}

type jobfile struct {
	srv.File
	reader jobreader
	writer jobwriter
}

// mkJob creates the subtree of files that represent a job in jobd and returns
// it to its caller.
func mkJob(root *srv.File, user p.User, def jobdef) (*job, error) {
	glog.V(4).Infof("Entering mkJob(%v, %v, %v)", root, user, def)
	defer glog.V(4).Infof("Exiting mkJob(%v, %v, %v)", root, user, def)

	glog.V(3).Infoln("Creating job directory: ", def.name)

	job := &job{defn: def, done: make(chan bool), history: ring.New(32)}

	ctl := &jobfile{
		// ctl reader returns the current state of the job.
		reader: func() []byte {
			return []byte(job.defn.state)
		},
		// ctl writer is responsible for stopping or starting the job.
		writer: func(data []byte) (int, error) {
			switch cmd := strings.ToLower(string(data)); cmd {
			case STOP:
				if job.defn.state != STOPPED {
					glog.V(3).Infof("Stopping job: %v", job.defn.name)
					job.defn.state = STOPPED
					job.done <- true
				}
				return len(data), nil
			case START:
				if job.defn.state != STARTED {
					glog.V(3).Infof("Starting job: %v", job.defn.name)
					job.defn.state = STARTED
					go job.run()
				}
				return len(data), nil
			default:
				return 0, fmt.Errorf("unknown command: %s", cmd)
			}
		}}
	if err := ctl.Add(&job.File, "ctl", user, nil, 0666, ctl); err != nil {
		glog.Errorf("Can't create %s/ctl [%v]", def.name, err)
		return nil, err
	}

	sched := &jobfile{
		// schedule reader returns the job's schedule and, if it's started, its
		// next scheduled execution time.
		reader: func() []byte {
			if job.defn.state == STARTED {
				e, _ := cronexpr.Parse(job.defn.schedule)
				return []byte(fmt.Sprintf("%s:%v", job.defn.schedule, e.Next(time.Now())))
			}
			return []byte(job.defn.schedule)
		},
		// schedule is read only.
		writer: func(data []byte) (int, error) {
			return 0, srv.Eperm
		}}
	if err := sched.Add(&job.File, "schedule", user, nil, 0444, sched); err != nil {
		glog.Errorf("Can't create %s/schedule [%v]", job.defn.name, err)
		return nil, err
	}

	cmd := &jobfile{
		// cmd reader returns the job's command.
		reader: func() []byte {
			return []byte(def.cmd)
		},
		// cmd is read only.
		writer: func(data []byte) (int, error) {
			return 0, srv.Eperm
		}}
	if err := cmd.Add(&job.File, "cmd", user, nil, 0444, cmd); err != nil {
		glog.Errorf("Can't create %s/cmd [%v]", job.defn.name, err)
		return nil, err
	}

	log := &jobfile{
		// log reader returns the job's execution history.
		reader: func() []byte {
			result := []byte{}
			job.history.Do(func(v interface{}) {
				if v != nil {
					for _, b := range bytes.NewBufferString(v.(string)).Bytes() {
						result = append(result, b)
					}
				}
			})
			return result
		},
		// log is read only.
		writer: func(data []byte) (int, error) {
			return 0, srv.Eperm
		}}
	if err := log.Add(&job.File, "log", user, nil, 0444, log); err != nil {
		glog.Errorf("Can't create %s/log [%v]", job.defn.name, err)
		return nil, err
	}

	return job, nil
}
```

Each of the job files is an instance of the `jobfile` type. A `jobfile` embeds the
go9p `srv.File` type and adds `reader`, and `writer` fields which are used by the
`jobfile.Read` and `jobfile.Write` methods. The directory that holds them all is an
instance of the `job` type. The code that `Add`s the
*ctl*, *schedule*, *cmd*, and *log* files to the subdirectory should be familiar by
now. The `Read`/`Write` behavior of each file is defined by the function
literals assigned to each file's reader/writer field.

To see how the `jobfile` `reader` and `writer` fields are used checkout the code from
job.go below.

```go
/*** job.go ***/
// Read handles read operations on a jobfile using its associated reader.
func (jf jobfile) Read(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	glog.V(4).Infof("Entering jobfile.Read(%v, %v, %)", fid, buf, offset)
	defer glog.V(4).Infof("Exiting jobfile.Read(%v, %v, %v)", fid, buf, offset)

	cont := jf.reader()

	if offset > uint64(len(cont)) {
		return 0, nil
	}

	contout := cont[offset:]

	copy(buf, contout)
	return len(contout), nil
}

// Write handles write operations on a jobfile using its associated writer.
func (jf *jobfile) Write(fid *srv.FFid, data []byte, offset uint64) (int, error) {
	glog.V(4).Infof("Entering jobfile.Write(%v, %v, %v)", fid, data, offset)
	defer glog.V(4).Infof("Exiting jobfile.Write(%v, %v, %v)", fid, data, offset)

	jf.Parent.Lock()
	defer jf.Parent.Unlock()

	return jf.writer(data)
}
```

The `Read` and `Write` methods contain the boiler plate and the reader/writer
fields supply the file specific behavior.

Oh, and last but not least there's the function which actually runs the jobs.

```go
/*** job.go ***/
// run executes the command associated with a job according to its schedule and
// records the results until it is told to stop.
func (j *job) run() {
	j.history.Value = fmt.Sprintf("%s:started\n", time.Now().String())
	j.history = j.history.Next()
	for {
		now := time.Now()
		e, err := cronexpr.Parse(j.defn.schedule)
		if err != nil {
			glog.Errorf("Can't parse %s [%s]", j.defn.schedule, err)
			return
		}

		select {
		case <-time.After(e.Next(now).Sub(now)):
			glog.V(3).Infof("running `%s`", j.defn.cmd)
			var out bytes.Buffer
			k := exec.Command("/bin/bash", "-c", j.defn.cmd)
			k.Stdout = &out
			if err := k.Run(); err != nil {
				glog.Errorf("%s failed: %v", j.defn.cmd, err)
				continue
			}
			glog.V(3).Infof("%s returned: %s", j.defn.name, out.String())
			j.history.Value = fmt.Sprintf("%s:%s", time.Now().String(), out.String())
			j.history = j.history.Next()
		case <-j.done:
			glog.V(3).Infof("completed")
			j.history.Value = fmt.Sprintf("%s:completed\n", time.Now().String())
			j.history = j.history.Next()
			return
		}
	}
}
```

It's a pretty straight forward go timer loop. You might be wondering if all the
code I've wrapped around it is worth the bother. Obviously I think it is
because once a machine has mounted the jobd file system anything that can read
and write files can schedule jobs on the jobd host.

