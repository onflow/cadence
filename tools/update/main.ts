import { Octokit } from "octokit"
import * as YAML from 'js-yaml'
import { readFile, mkdtemp, rm } from 'node:fs/promises'
import {CadenceUpdateToolConfigSchema, Mod, Repo} from './config.schema'
import * as yargs from 'yargs'
import * as path from "path"
import axios from "axios"
import {SemVer, valid as asValidSemVer} from "semver"
import * as octokitTypes from '@octokit/openapi-types'
import { RequestError } from "@octokit/request-error"
import * as os from "os"
import exec from "executive"
const prompts = require('prompts')

type Pull = octokitTypes.components["schemas"]['pull-request-simple']

function isValidSemVer(version: string): boolean {
    return !!asValidSemVer(version)
}

function semVerAtLeast(actualVersion: string, expectedVersion: string): boolean {
    return isValidSemVer(actualVersion)
        && isValidSemVer(expectedVersion)
        && new SemVer(actualVersion).compare(expectedVersion) >= 0
}

function extractVersionCommit(version: string): string | null {
    const parts = version.split('-')
    if (parts.length <= 1) {
        return null
    }
    return parts[parts.length-1]
}

class Updater {

    constructor(
        public versions: Map<string, string>,
        public config: CadenceUpdateToolConfigSchema,
        public octokit: Octokit
    ) {}

    async update(repoName: string | undefined): Promise<void> {
        const rootFullRepoName = this.config.repo
        const rootRepoVersion = this.getExpectedVersion(rootFullRepoName)

        const reposDescription = repoName === undefined ? 'all repos': repoName
        console.log(`Updating ${reposDescription} to ${rootFullRepoName} version ${rootRepoVersion}`)

        for (const repo of this.config.repos) {
            if (repoName !== undefined && repo.repo !== repoName) {
                continue
            }

            if (!await this.updateRepo(repo))
                break
        }
    }

    async updateRepo(repo: Repo): Promise<boolean> {
        const fullRepoName = repo.repo
        console.log(`Checking repo ${fullRepoName} ...`)

        const expectedVersion = this.versions.get(fullRepoName)
        if (expectedVersion) {
            if (await this.repoModsUpdated(expectedVersion, repo)) {
                return true
            }
        }

        const latestReleaseTagName = await this.fetchLatestReleaseTagName(fullRepoName)
        if (latestReleaseTagName !== null) {
            console.log(`Latest release of repo ${fullRepoName}: ${latestReleaseTagName}`)

            if (await this.repoModsUpdated(latestReleaseTagName, repo)) {
                return true
            }

            console.log(`Latest release of repo ${fullRepoName} (${latestReleaseTagName}) is not updated, checking default branch ...`)
        } else {
            console.log(`Checking default branch ...`)
        }

        const [owner, repoName] = fullRepoName.split('/')

        const repoResponse = await this.octokit.rest.repos.get({owner, repo: repoName})
        const defaultBranch = repoResponse.data.default_branch
        console.log(`Default branch of repo ${fullRepoName}: ${defaultBranch}`)

        const defaultRefResponse = await this.octokit.rest.git.getRef({
            owner,
            repo: repoName,
            ref: `heads/${defaultBranch}`
        })
        const defaultRef = defaultRefResponse.data.object.sha

        if (await this.repoModsUpdated(defaultRef, repo)) {
            console.log(`Default branch (${defaultBranch}) of repo ${fullRepoName} is updated`)

            if (repo.needsRelease) {
                await this.release(repo, defaultBranch)

                return false
            }

            return true
        }

        if (await this.repoHasUpdatePR(repo)) {
            console.log(`Update PR needs to get merged`)
            return false
        }

        const updateAnswer = await prompts({
            type: 'confirm',
            name: 'updateRepo',
            message: 'Would you like to update the repo and create a PR?'
        })

        if (updateAnswer.updateRepo) {
            await this.updateRepoModsInPR(repo)
        }

        return false
    }

    private async release(repo: Repo, defaultBranch: string) {
        const fullRepoName = repo.repo
        const [owner, repoName] = fullRepoName.split('/')

        for (const mod of repo.mods) {

            if (mod.path === '') {
                console.log(`Repo ${fullRepoName} should be released`)
            } else {
                console.log(`Repo ${fullRepoName} mod ${mod.path} should be released`)
            }

            const createAnswer = await prompts({
                type: 'confirm',
                name: 'createRelease',
                message: 'Would you like to create a release?'
            })

            if (createAnswer.createRelease) {
                const versionAnswer = await prompts({
                    type: 'text',
                    name: 'version',
                    message: 'Version:',
                    validate: (value: string) => value.trim().length > 0
                })

                const version = versionAnswer.version.trim()

                await new Releaser(fullRepoName, mod.path, version, this.octokit).release()
            }
        }
    }

    async repoModsUpdated(refName: string, repo: Repo): Promise<boolean> {
        const fullRepoName = repo.repo
        console.log(`Checking if all mods of repo ${fullRepoName} at version ${refName} are updated ...`)

        for (const mod of repo.mods) {
            if (!await this.repoModUpdated(refName, repo, mod)) {
                return false
            }
        }

        console.log(`All mods of mod ${fullRepoName} at repo version ${refName} are up-to-date`)

        return true
    }

    // repoModUpdated returns true of the given repo's mod
    //
    async repoModUpdated(refName: string, repo: Repo, mod: Mod): Promise<boolean> {
        const fullRepoName = repo.repo
        const fullModName = path.join(fullRepoName, mod.path)
        console.log(`Checking if mod ${fullModName} at repo version ${refName} is updated ...`)

        const goMod = await Updater.fetchRaw(fullRepoName, refName, path.join(mod.path, "go.mod"))

        for (const dep of mod.deps) {
            const matches = goMod.match(new RegExp(`${dep} ([^ /\n]+)`))
            if (matches === null) {
                console.error(`Missing go.mod entry for dep ${dep}`)
                process.exit(1)
            }

            const expectedVersion = this.getExpectedVersion(dep)
            const actualVersion = matches[1]

            const actualVersionCommit = extractVersionCommit(actualVersion)

            if (!(semVerAtLeast(actualVersion, expectedVersion) || actualVersionCommit == expectedVersion)) {
                console.warn(`Outdated dep ${dep}: expected ${expectedVersion}, got ${actualVersionCommit ?? actualVersion}`)
                return false
            }
        }

        const refNameParts = refName.split('/')
        const version = refNameParts[refNameParts.length-1]

        console.log(`All deps of mod ${fullModName} at repo version ${version} are up-to-date`)

        this.versions.set(fullModName, version)

        return true
    }

    // fetchLatestReleaseTagName fetches the latest release tag name for a given repo
    //
    async fetchLatestReleaseTagName(fullRepoName: string): Promise<string | null> {
        try {
            const [owner, repoName] = fullRepoName.split('/')
            const release = await this.octokit.rest.repos.getLatestRelease({
                owner,
                repo: repoName,
            })
            return release.data.tag_name
        } catch (e) {
            if (e instanceof RequestError) {
                if (e.status === 404) {
                    return null
                }
            }
            throw e
        }
    }

    // fetchRaw fetches the given path of the given repo, at the given version/tag name
    //
    static async fetchRaw(fullRepoName: string, refName: string, path: string): Promise<string> {
        const response = await axios.get(`https://raw.githubusercontent.com/${fullRepoName}/${refName}/${path}`)
        return response.data
    }

    // repoHasUpdatePR returns true if the given repo has an update PR open.
    // See prIsUpdate for the definition of an update PR.
    //
    async repoHasUpdatePR(repo: Repo): Promise<boolean> {
        console.log(`Checking if an update PR exists ...`)

        const fullRepoName = repo.repo
        const [owner, repoName] = fullRepoName.split('/')

        for await (const page of this.octokit.paginate.iterator(
            this.octokit.rest.pulls.list,
            {
                owner,
                repo: repoName,
                state: "open"
            }
        )) {
            for (const pull of page.data) {
                if (this.prIsUpdate(pull, repo)) {
                    return true
                }
            }
        }

        console.log(`No update PR found`)

        return false
    }

    // prIsUpdate returns true if the given PR is an update PR.
    //
    // An update PR is any PR which mentions "update" (case-insensitive),
    // and updates at least one of the deps,
    // by mentioning the dependency/version pair in the description
    //
    prIsUpdate(pull: Pull, repo: Repo): boolean {
        console.log(`Checking if PR ${pull.number} updates a dep of a mod ...`)

        if (!pull.title.match(/[uU]pdate/)) {
            return false
        }

        const expectedVersions = new Map<string, string>()
        for (const mod of repo.mods) {
            for (const dep of mod.deps) {
                if (!expectedVersions.has(dep)) {
                    expectedVersions.set(
                        dep,
                        this.getExpectedVersion(dep)
                    )
                }
            }
        }

        for (const [dep, expectedVersion] of expectedVersions.entries()) {
            console.log(`Checking if PR ${pull.number} updates dep ${dep} to ${expectedVersion} ...`)

            if ((pull.body?.indexOf(`${dep} ${expectedVersion}`) ??  -1) >= 0) {
                console.log(`PR ${pull.html_url} updates dep ${dep} to ${expectedVersion}`)

                return true
            }
        }

        console.log(`PR ${pull.html_url} is not an update PR`)

        return false
    }

    // getExpectedVersion returns the version which should be used for the given repo.
    //
    getExpectedVersion(fullModName: string): string {
        const expectedVersion = this.versions.get(fullModName)
        if (!expectedVersion) {
            console.error(`Missing version for ${fullModName}`)
            process.exit(1)
        }
        return expectedVersion
    }

    // updateRepoModsInPR updates the deps of the mods in the given repo and creates a PR.
    // It clones the repo, creates a branch, updates the dependencies, commits the changes,
    // and creates a PR for the branch.
    //
    async updateRepoModsInPR(repo: Repo): Promise<void> {
        const fullRepoName = repo.repo
        const [owner, repoName] = fullRepoName.split('/')
        const dir = await mkdtemp(path.join(os.tmpdir(), `${owner}-${repoName}`))

        console.log(`Cloning ${fullRepoName} ...`)
        await exec(`git clone git@github.com:${fullRepoName} ${dir}`)
        process.chdir(dir)

        const rootFullRepoName = this.config.repo
        const [rootRepoOwner, rootRepoName] = rootFullRepoName.split('/')
        const rootRepoVersion = this.getExpectedVersion(rootFullRepoName)
        const branch = ['auto-update', rootRepoOwner, rootRepoName, rootRepoVersion].join('-')
        console.log(`Creating branch ${branch} ...`)
        await exec(`git checkout -b ${branch}`)

        // TODO: only update dependencies that are updatable

        const updates = new Map<string, string>()
        for (const mod of repo.mods) {
            const fullModName = path.join(fullRepoName, mod.path)
            console.log(`Updating mod ${fullModName} ...`)
            process.chdir(path.join(dir, mod.path))

            for (const dep of mod.deps) {
                const newVersion = this.getExpectedVersion(dep)
                console.log(`Updating mod ${fullModName} dep ${dep} to version ${newVersion} ...`)
                await exec(`go get github.com/${dep}@${newVersion}`)
                updates.set(dep, newVersion)
            }

            console.log(`Cleaning up mod ${fullModName} ...`)
            await exec(`go mod tidy`)
        }

        console.log(`Committing update ...`)
        await exec(`git commit -a -m "auto update to ${rootFullRepoName} ${rootRepoVersion}"`)

        console.log(`Pushing update ...`)
        await exec(`git push -u origin ${branch}"`)

        console.log(`Creating PR ...`)

        let updateList = ''
        for (const [dep, version] of updates.entries()) {
            updateList += `- [${dep} ${version}](https://github.com/${dep}/releases/tag/${version})\n`
        }

        const pull = await this.octokit.rest.pulls.create({
            owner,
            repo: repoName,
            head: branch,
            base: 'master',
            title: `Auto update to ${rootFullRepoName} ${rootRepoVersion}`,
            body: `
## Description

Automatically update to:
${updateList}
`,
        })

        const {html_url: prURL, number: prNumber} = pull.data

        console.log(`Created PR ${prURL}`)

        const { updateLabels } = repo
        if (updateLabels) {
            console.log(`Adding labels to PR ${prURL}: ${updateLabels}`)
            await this.octokit.rest.issues.addLabels({
                owner,
                repo: repoName,
                issue_number: prNumber,
                labels: updateLabels
            })
        }

        console.log(`Cleaning up clone of ${fullRepoName} ...`)
        await rm(dir, { recursive: true, force: true })
    }

}

async function authenticate(): Promise<Octokit> {
    const octokit = new Octokit({
        auth: process.env.GH_TOKEN
    })

    const {data: {login}} = await octokit.rest.users.getAuthenticated()
    console.log("👋 Hello, %s", login)

    return octokit
}

function getTagName(modPath: string, version: string) {
    if (modPath === '') {
        return version
    }

    let tag = modPath
    if (tag[tag.length - 1] !== '/') {
        tag += '/'
    }
    return tag + version
}

class Releaser {

    constructor(
        public repo: string,
        public modPath: string,
        public version: string,
        public octokit: Octokit
    ) {}

    // release tags a release of the given repo.
    // It clones the repo, creates a tag, and pushes the tag.
    //
    async release(): Promise<void> {
        const [owner, repoName] = this.repo.split('/')

        // Check if tag already exists
        const tag = getTagName(this.modPath, this.version)

        try {
            await this.octokit.rest.git.getRef({
                owner,
                repo: repoName,
                ref: `tags/${tag}`
            })

            console.log(`Tag already exists! Create a GitHub release: https://github.com/${this.repo}/releases/new?tag=${tag}`)
            return
        } catch (e) {}

        const dir = await mkdtemp(path.join(os.tmpdir(), `${owner}-${repoName}`))

        console.log(`Cloning ${this.repo} ...`)
        await exec(`git clone git@github.com:${this.repo} ${dir}`)
        process.chdir(dir)

        console.log(`Tagging ${this.repo} version ${this.version} ...`)
        await exec(`git tag ${tag}`)

        if (this.modPath !== '') {
            console.log(`Pushing ${this.repo} mod ${this.modPath} version ${this.version} ...`)
        } else {
            console.log(`Pushing ${this.repo} version ${this.version} ...`)
        }
        await exec(`git push origin --tags`)

        console.log(`Cleaning up clone of ${this.repo}`)
        await rm(dir, { recursive: true, force: true })

        console.log(`Now create a GitHub release: https://github.com/${this.repo}/releases/new?tag=${tag}`)
    }
}

(async () => {
    const octokit = await authenticate()

    await yargs
        .version(false)
        .command(
            "update",
            "update, if needed",
            {
                version: {
                    alias: 'v',
                    type: 'string',
                    describe: 'The version to update to',
                    demandOption: true
                },
                repo: {
                    alias: 'r',
                    type: 'string',
                    describe: 'The repo name',
                },
                config: {
                    alias: 'c',
                    type: 'string',
                    describe: 'The path of the config',
                    default: 'config.yaml'
                },
                versions: {
                    type: 'string',
                    describe: 'Comma separated list of repo@version',
                    default: ''
                }
            },
            async (args) => {
                const configData = await readFile(args.config, 'utf8')
                const config = YAML.load(configData) as CadenceUpdateToolConfigSchema

                const versions = new Map([
                    [config.repo, args.version],
                ])

                for (const versionEntry of args.versions.split(',')) {
                    const [repo, version] = versionEntry.split('@')
                    versions.set(repo, version)
                }

                await new Updater(versions, config, octokit)
                    .update(args.repo)
            }
        )
        .command(
            "release",
            "release a repo",
            {
                'version': {
                    alias: 'v',
                    type: 'string',
                    describe: 'The version name',
                    demandOption: true
                },
                'repo': {
                    alias: 'r',
                    type: 'string',
                    describe: 'The repo name',
                    demandOption: true
                },
                'mod': {
                    alias: 'm',
                    type: 'string',
                    describe: 'The mod path'
                },
            },
            async (args) => {
                await new Releaser(args.repo, args.mod || '', args.version, octokit).release()
            }
        )
        .parse()
})()
