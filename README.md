# Zond

[![CircleCI](https://circleci.com/gh/theQRL/zond.svg?style=shield)](https://circleci.com/gh/theQRL/zond)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/958f22ba2f404e4bb45b8fef1a8ad1e5)](https://www.codacy.com/gh/theQRL/zond/dashboard?utm_source=github.com&utm_medium=referral&utm_content=theQRL/zond&utm_campaign=Badge_Grade)

> *WARNING: This is work in progress, NOT FOR PRODUCTION USE*

POS QRL Node Implementation written in GO. This code is still under heavy development and is currently undergoing DEVNET testing internally.

Zond will undergo an additional public testnet once the code stabilizes and we are complete with integration and development. Look for a request for testing in our public channels.

## Installing

Updated and upgraded Ubuntu distribution with `aes-ni` for cryptographic support and ample HDD space to store the chain is required.

### Ubuntu

#### Dependencies

##### Bazel

> [Bazel](https://www.bazel.build/) is an open-source build and test tool similar to Make

Install Bazel in Ubuntu following [their official instructions](https://docs.bazel.build/versions/master/install-ubuntu.html) with:

```bash
# Add Bazel distribution URI as a package source
sudo apt install curl gnupg
curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor > bazel.gpg
sudo mv bazel.gpg /etc/apt/trusted.gpg.d/
echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list

# Update and Install 
sudo apt update && sudo apt install bazel
```

##### Zond Repo

Clone this repo to the local file system

```bash
git clone https://github.com/theQRL/zond.git
```

#### Build zond

Using the Bazel build tools build both `zond` and `zond-cli`

```bash
cd zond

bazel build //cmd/zond:zond
bazel build //cmd/zond-cli:zond-cli
```

You now have the node and node CLI installed in `zond/bazel-bin/cmd/`

## Running zond

> Work In Progress to be released when public Testnet is live!

## Support

Questions? Issues?

Please come ask in our Discord Server: <https://discord.gg/qrl>
