NCAA Push Utility
==================

Command line utility to perform the final steps needed to push code to the staging server where it will await deployment. This involves merging of a topic branch into master, tagging a new version, and updating the site makefile.

You must provide two locations for the utility to work. These can be persisted as environment variables or passed into the utility as options every time you run it:

* Absolute path to site repo (*--site-repo* or *NCAA_BARCA_SITE_REPO_PATH*)
* \*.make filename (*--site-makefile* or *NCAA_BARCA_SITE_MAKEFILE*)

It is recommended that you use environment variables as these values are not likely to change once your development environment is setup:

```bash
export NCAA_BARCA_SITE_REPO_PATH=/Users/mstills/Repos/barcelona/master
export NCAA_BARCA_SITE_MAKEFILE=barcelona.make
```

Installation
============
Once you've cloned the repo, change directory to it and compile the program:

1. ```$ go build ncaapushit.go``` (make sure you have [https://golang.org/dl/](Go installed already))
2. Add it to your PATH or run "./ncaapushit" to run the utility.
3. Optionally set the environment variables described above.

Usage
=====
The ```ncaapushit``` utility is expected to be used in the following manner:

1. Merge your pull request (NCAA-XXXX -> master) in the web interface
2. In your terminal, change directories to your local repo
3. You should be checked out to ```NCAA-XXXX``` branch that is ready to queue for deployment. This repo should be in a clean working state (no pending changes, as they should have been committed already before the pull requeste was merged).
4. Run the command, specifying the version column you want to bump (by default, the "patch" or least significant  column will be bumped):

```bash
$ ncaapushit --bump="minor"
```

There are a variety of other options that you might find useful:

```bash
$ ncaapushit --help
```

The utility will then perform the following steps assuming there are no problems along the way:

1. Update local repos (site and module)
2. Clean up (delete) merged topic branch as it is no longer needed
3. Ask for you to review the new version vs. the old version
4. Create a tag in the local repo for the new version and push it up to the remote.
5. Put the new tag into the makefile in the site repo
6. Format a commit message and make the commit
7. Push the site repo changes in order to trigger a staging build.

This utility should never leave your work in a damaged state. If it fails, it is expected to fail gracefully. If you have any problems with this utility, please report them to Matt Stills.