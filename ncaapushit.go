// NCAA Pushit (version Barcelona-0.0.1)
// by Matt Stills <matthew.stills@turner.com>
//
// Intended to be run after merging an NCAA Barcelona pull request via
// Bitbucket. This utility will bump and tag a new version of a given
// repo, insert the new version tag into the site makefile, and commit
// these changes with a properly formatted commit message to the app.
//
// You may set the following environment variables to avoid having to
// pass options for these values each time you use the utility:
//
// NCAA_BARCA_SITE_REPO_PATH (default = "~/Repos/ncaa-barcelona")
// NCAA_BARCA_SITE_MAKEFILE  (default = "barcelona.make")
package main

import (
    "bufio"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "os/user"
    "strconv"
    "strings"
)

type nestedMap map[string]map[string]string
type gitc []string
type pushError struct {
    msg string
}

var (
    // options for this utility
    bumpOpt     string
    moduleOpt   string
    siteRepoOpt string
    siteMakeOpt string
    topicOpt    string
    // cwd or overridden module dir
    cwd string
)

var usr, _ = user.Current()
var optionsMap = nestedMap{
    "bump": {
        "usage":   "The semver column of the module version to bump (major|minor|patch).",
        "default": "patch",
    },
    "module": {
        "usage":   "The path to the module with changes to push.",
        "default": "$PWD",
    },
    "site-repo": {
        "usage":   "The path to your site (app) repo where the makefile resides.",
        "default": usr.HomeDir + "/Repos/ncaa-barcelona",
    },
    "site-makefile": {
        "usage":   "Filename of the *.make file to alter.",
        "default": "barcelona.make",
    },
    "topic": {
        "usage": "If you have already merged your topic branch, you must provide the name of it (eg. NCAA-31337), otherwise the current branch will be used.",
    },
}

var gitCommands = map[string]gitc{
    "update":   {"up"},
    "branch":   {"rev-parse", "--abbrev-ref", "HEAD"},
    "latest":   {"describe", "master", "--abbrev=0", "--tags"},
    "coMaster": {"checkout", "master"},
    "pushit":   {"push", "origin", "master"},
    "pushtags": {"push", "origin", "--tags"},
}

// error reporter/handler for the utility
func (e *pushError) Error() string {
    return fmt.Sprintf("\nfatal: %s", e.msg)
}

// applyEnvOptions checks whether or not site options are using defaults. If so,
// it attemps to use environment variables to set them instead.
func applyEnvOptions() {
    if siteRepoOpt == optionsMap["site-repo"]["default"] {
        if envRepo := os.Getenv("NCAA_BARCA_SITE_REPO_PATH"); envRepo != "" {
            siteRepoOpt = envRepo
        }
    }

    if siteMakeOpt == optionsMap["site-makefile"]["default"] {
        if envMake := os.Getenv("NCAA_BARCA_SITE_MAKEFILE"); envMake != "" {
            siteMakeOpt = envMake
        }
    }
}

// getMakefile reads the provided site directory and locates the makefile
func getMakefile() (string, error) {
    var makefile string

    siteFiles, err := ioutil.ReadDir(siteRepoOpt)
    foundMakefile := false

    if err != nil {
        return "", &pushError{("There was a problem reading the site repo directory @ " + siteRepoOpt)}
    }

    for _, file := range siteFiles {
        if file.Name() == siteMakeOpt {
            foundMakefile = true
            break
        }
    }

    if !foundMakefile {
        return "", &pushError{("Could not locate makefile @ '" + siteRepoOpt + "/" + siteMakeOpt + "'")}
    }

    makefile = siteRepoOpt + "/" + siteMakeOpt

    return makefile, nil
}

// getModule determines the current module name from the current working path
func getModule() (string, error) {
    var module string

    // $PWD (the default) instructs us to get the current working dir
    if moduleOpt == "$PWD" {
        cwd, _ = os.Getwd()
    } else {
        cwd = moduleOpt
    }

    // we obtain the module name from the last element of the path
    cwdParts := strings.Split(cwd, string(os.PathSeparator))
    module = string(cwdParts[len(cwdParts)-1])

    // verify that the dir exists and has a *.module within
    files, readErr := ioutil.ReadDir(cwd)
    foundModule := false

    if readErr != nil {
        return "", &pushError{("There was a problem reading the module directory @ " + cwd + "\n\nPlease change directory to the top-level of the module repo you want to act on (ie. where the *.module file is located) and try again.\nYou may provide a full path using the '--module' option of this utility.\n")}
    }

    // change to the provided directory if we're not already there
    if moduleOpt != "$PWD" {
        os.Chdir(cwd)
    }

    for _, file := range files {
        if seekModule := module + ".module"; seekModule == file.Name() {
            foundModule = true
            break
        }
    }

    if !foundModule {
        return "", &pushError{("Could not locate module '" + module + "' @ " + cwd)}
    }

    fmt.Println("Module repo:", module)

    return module, nil
}

// git runs a git command in given directory
func git(command gitc, dir string) []byte {
    os.Chdir(dir)
    out, err := exec.Command("git", command...).CombinedOutput()

    if err != nil {
        fmt.Println(string(out))
        panic("there was a problem running the git command '" + strings.Join(command, " ") + "'. See output above for clues.")
    }

    return out
}

// tagVersion creates the new tag in Git and pushes it to site repo (origin)
func tagVersion(version string, complete chan bool) {
    // if module repo was not checked out to master already, perform clean up and prepare for tagging
    if topicOpt != "master" {
        git(gitCommands["coMaster"], cwd)        // checkout master
        git(gitc{"branch", "-d", topicOpt}, cwd) // delete topic branch which we assume has been merged via pull request

        fmt.Printf("Module Repo Cleanup: Local topic branch '%s' was deleted.\n", topicOpt)
    }

    git(gitc{"tag", "v" + version}, cwd)
    git(gitCommands["pushtags"], cwd)

    complete <- true
}

// getVersions determines the latest module version (via Git) and bumps the appropriate semver column
func getVersions() (string, string, error) {
    var (
        newVersion    [3]int
        currentBranch string
        latest        string
        splitVersion  []string
    )

    fmt.Print("\t`-- updating...")
    git(gitCommands["update"], cwd)
    fmt.Print(" complete\n")

    currentBranch = string(git(gitCommands["branch"], cwd))
    currentBranch = strings.Trim(currentBranch, " \n\t\r")

    if currentBranch == "master" && topicOpt == "" {
        return "", "", &pushError{"If you have already merged your branch, you must provide it via the --topic option. Otherwise, checkout the branch and re-run this utility."}
    }

    if topicOpt != "" && currentBranch != topicOpt && currentBranch != "master" {
        return "", "", &pushError{"The branch supplied via --topic does not match the current module branch (" + topicOpt + " != " + currentBranch + ")"}
    }

    // if no topic was supplied, store the current branch for future reference
    if topicOpt == "" {
        topicOpt = currentBranch
    }

    // ** get the latest tag and bump it
    gitVer := git(gitCommands["latest"], cwd)

    latest = strings.Trim(string(gitVer[1:]), " \n\t")
    fmt.Printf("Current version: %s\n", latest)
    splitVersion = strings.Split(latest, ".")

    switch bumpOpt {
    case "major":
        newVersion[0], _ = strconv.Atoi(splitVersion[0])
        newVersion[0]++

        splitVersion[0] = strconv.Itoa(newVersion[0])
        splitVersion[1] = "0"
        splitVersion[2] = "0"
        break
    case "minor":
        newVersion[1], _ = strconv.Atoi(splitVersion[1])
        newVersion[1]++

        splitVersion[1] = strconv.Itoa(newVersion[1])
        splitVersion[2] = "0"
        break
    case "patch":
        newVersion[2], _ = strconv.Atoi(splitVersion[2])
        newVersion[2]++

        splitVersion[2] = strconv.Itoa(newVersion[2])
        break
    }

    return strings.Join(splitVersion, "."), latest, nil
}

// getUpdatedMakefile scans existing makefile for current module + version, replaces that line with the new version
func getUpdatedMakefile(makefile, module, newVersion, latest string) ([]string, error) {
    var outFile []string

    file, _ := os.Open(makefile)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    seekLine := ("projects[" + module + "][download][tag] = \"v" + string(latest) + "\"")
    replacedVersion := false

    // read the makefile in line by line using the scanner
    for scanner.Scan() {
        if strings.Contains(scanner.Text(), seekLine) {
            // update the version once the correct line is located
            replaceVersion := strings.Replace(scanner.Text(), strings.Trim(string(latest), "\n\t "), newVersion, -1)
            outFile = append(outFile, replaceVersion)

            replacedVersion = true
        } else {
            outFile = append(outFile, scanner.Text())
        }
    }

    if !replacedVersion {
        return outFile, &pushError{"Either the module '" + module + "' or latest tag 'v" + latest + "' was not found in the makefile.\nMake sure your site repo is up-to-date before using this utility."}
    }

    return outFile, nil
}

// pushUpdatedMakefile writes the new makefile contents to disk, commits the change, and pushes it up to the site repo
func pushUpdatedMakefile(outFile *[]string, commitMsg string, taggingComplete chan bool) error {
    fmt.Println(commitMsg)
    fmt.Println("\t`-- committed changes with message")

    // make sure this repo is up to date and checked out to master
    git(gitCommands["update"], siteRepoOpt)
    git(gitCommands["coMaster"], siteRepoOpt)

    // write the updated makefile
    writeFile := []byte(strings.Join(*outFile, "\n"))
    err := ioutil.WriteFile(siteRepoOpt+"/"+siteMakeOpt, writeFile, 0644)

    if err != nil {
        return &pushError{"Could not write new makefile. Check permissions and try again."}
    }

    // commit the changes and pushit
    git(gitc{"commit", siteMakeOpt, "-m", commitMsg}, siteRepoOpt)

    // wait for tagging to finish before pushing
    <-taggingComplete

    git(gitCommands["pushit"], siteRepoOpt)

    return nil
}

func init() {
    // option: --bump / -v
    flag.StringVar(&bumpOpt, "bump", optionsMap["bump"]["default"], optionsMap["bump"]["usage"])
    flag.StringVar(&bumpOpt, "v", optionsMap["bump"]["default"], "shorthand for --bump")

    // option: --module
    flag.StringVar(&moduleOpt, "module", optionsMap["module"]["default"], optionsMap["module"]["usage"])

    // option: --site-repo / -r
    flag.StringVar(&siteRepoOpt, "site-repo", optionsMap["site-repo"]["default"], optionsMap["site-repo"]["usage"])
    flag.StringVar(&siteRepoOpt, "r", optionsMap["site-repo"]["default"], "shorthand for --site-repo")

    // option: --site-makefile
    flag.StringVar(&siteMakeOpt, "site-makefile", optionsMap["site-makefile"]["default"], optionsMap["site-makefile"]["usage"])

    // option: --topic
    flag.StringVar(&topicOpt, "topic", optionsMap["topic"]["default"], optionsMap["topic"]["usage"])
}

func main() {
    var (
        module     string
        makefile   string
        latest     string
        newVersion string
        outFile    []string
        err        error
    )

    taggingComplete := make(chan bool)

    flag.Parse()      // handle options passed in via command-line
    applyEnvOptions() // try environment variables for missing options

    // ** make sure a valid module option has been provided
    module, err = getModule()

    if err != nil {
        fmt.Println(err)
        return
    }

    // ** make sure a valid makefile can be found in the site repo directory
    makefile, err = getMakefile()

    if err != nil {
        fmt.Println(err)
        return
    }

    // ** perform various git tasks, get the new version back
    newVersion, latest, err = getVersions()

    if err != nil {
        fmt.Println(err)
        return
    }

    // ** make sure the user is satisfied with the new version that will be tagged
    reader := bufio.NewReader(os.Stdin)

    fmt.Println("New version:", newVersion)
    fmt.Printf("Are you sure you want to tag and push this new version to staging? (y/n): ")

    text, _ := reader.ReadString('\n')
    text = strings.Trim(text, "\n")

    if text != "y" {
        fmt.Println("Aborting...")
        return
    }

    // while the rest proceeds, we can go ahead and start pushing the new tag up from the module repo
    go tagVersion(newVersion, taggingComplete)

    outFile, err = getUpdatedMakefile(makefile, module, newVersion, latest)

    if err != nil {
        fmt.Println(err)
        return
    }

    commitMsg := fmt.Sprintf("\n%s %s -> %s", topicOpt, module, newVersion)
    err = pushUpdatedMakefile(&outFile, commitMsg, taggingComplete) // pass the tagging channel to make sure this stays in sync

    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Println("\nPush completed successfully!\nYour new version will build to the staging environment momentarily.")
}