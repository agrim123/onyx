# onyx

> Still work in progress. Improvements, suggestions and criticisms are welcome.

Onyx is a small wrapper over AWS SDK to perform some console ui tasks easily via command line!

## Install

### Direct use

- Download the [latest release](https://github.com/agrim123/onyx) binary for your operating system and you are good to go!

### Build from source

- Clone the repository
- Run `make install`. If your `GOPATH` is correctly set, this should give you a global alias `onyx` to use!

### Requirements

Onyx requires you have configured your aws keys in local using `aws configure` and have proper permissions to perform the required actions. 

You can install aws cli [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html).

### Usage

A quick `onyx` displays the available commands. Current supported namespaces are:
- ec2
    - security groups
- ecs
- iam
    - get user
