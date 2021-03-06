# Mantle: Gluing CoreOS together

This repository is a collection of utilities for the CoreOS SDK.

## plume

CoreOS release utility

## plume index

Generate and upload index.html objects to turn a Google Cloud Storage
bucket into a publicly browsable file tree. Useful if you want something
like Apache's directory index for your software download repository.

## kola

Kola is a framework for testing software integration in CoreOS instances
across multiple platforms. It is primarily designed to operate within
the CoreOS SDK for testing software that has landed in the OS image.
Ideally, all software needed for a test should be included by building
it into the image from the SDK.

Kola supports running tests on multiple platforms, currently QEMU and
GCE. In the future systemd-nspawn and EC2 may be added. Local platforms
do not rely on access to the Internet as a design principal of kola.
Tests that do so will break on local platforms.

Kola is still under heavy development and it is expected that its
interface will continue to change. Both the CLI and test registration
interface have upcoming changes.

### kola run
The run command invokes the main kola test harness. The harness will
run any registered tests on all platforms unless otherwise specified. It
runs any tests whose registered names matches a glob pattern.

`kola run <glob pattern>`

### kola test registration
Registering kola tests currently requires that the tests are registered
under the kola package and that the test function itself lives within
the mantle codebase.

Groups of similar tests are registered in an init() function inside the
kola package.  `Register(*Test)` is called per test. A kola `Test`
struct requires a unique name, and a single function that is the entry
point into the test.

### kola test writing
A kola test is a go function that is passed a `platform.TestCluster` to
run code against.  Its signature is `func(platform.TestCluster) error`
and must be registered and built into the kola binary. 

A `TestCluster` implements the `platform.Cluster` interface and will
give you access to a running cluster of CoreOS machines. A test writer
can interact with these machines through this interface.

To see test examples look under
[kola/tests](https://github.com/coreos/mantle/tree/master/kola/tests) in the
mantle codebase.

### kola native code
For some tests, the `Cluster` interface is limited and it is desirable to
run native go code directly on one of the CoreOS machines. This is
currently possible by using the `NativeFuncs` field of a kola `Test`
struct. This like a limited RPC interface.

`NativeFuncs` is used similar to the `Run` field of a registered kola
test. It registers and names functions in nearby packages.  These
functions, unlike the `Run` entry point, must be manually invoked inside
a kola test using a `TestCluster`'s `RunNative` method. The function
itself is then run natively on the specified running CoreOS instances.

For more examples, look at the
[coretest](https://github.com/coreos/mantle/tree/master/kola/tests/coretest)
suite of tests under kola. These tests were ported into kola and make
heavy use of the native code interface.

## ore

Related commands to launch instances on Google Compute Engine(gce)
within the latest SDK image. SSH keys should be added to the gce project
metadata before launching a cluster. All commands have flags that can
overwrite the default project, bucket, and other settings.  `ore help
<command>` can be used to discover all the switches.

### ore upload

Upload latest SDK image to Google Storage and then create a gce image.
Assumes an image packaged with the flag `--format=gce` is present.
Common usage for CoreOS devs using the default bucket and project is:

`ore upload`

### ore list-images

Print out gce images from a project. Common usage:

`ore list-images`

### ore create-instances

Launch instances on gce. SSH keys should be added to the metadata
section of the gce developers console. Common usage:

`ore create-instances -n=3 -image=<gce image name> -config=<cloud config file>`

### ore list-instances

List running gce instances. Common usage:

`ore list-instances`

### ore destroy-instances

Destroy instances on gce based by prefix or name. Images created with
`create-instances` use a common basename as a prefix that can also be
used to tear down the cluster. Common usage:

`ore destroy-instances -prefix=$USER`

