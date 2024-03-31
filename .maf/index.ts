import { spawnFn } from "@maf/core.ts";
import { run } from "@maf/docker/mod.ts";
import { getGoBuildEnv, GOARCH, GoBuild, GOOS } from "@maf/lang/go.ts";
import { getClient, Release, webhook } from "@maf/service/github.ts";
import { format as formatBytes } from "@std/fmt/bytes.ts";

const matrix = [
  { os: GOOS.linux, arch: GOARCH.amd64 },
  { os: GOOS.linux, arch: GOARCH.arm64 },
  { os: GOOS.windows, arch: GOARCH.amd64 },
];

export async function release(name: string, tag: string) {
  const client = await getClient();
  if (client === null) throw new Error(`no github access`);
  await client.createRelease("b1naryth1ef/carto", {
    name,
    tag,
  });
}

export async function buildAll() {
  await Promise.all(matrix.map((it) => build({ opts: { go: it } })));
}

export async function build({ opts, release, sha }: {
  opts?: { go: GoBuild; version?: string };
  release?: Release;
  sha?: string;
}) {
  const client = await getClient();

  const go = opts?.go || { os: GOOS.linux, arch: GOARCH.amd64 };
  let name = `carto-${go.os}-${go.arch}`;
  if (go.os === "windows") {
    name = name + ".exe";
  }

  if (client && sha) {
    await client.createCommitStatus("b1naryth1ef/carto", sha, {
      state: "pending",
      context: `carto-${go.os}-${go.arch}`,
    });
  }

  await run(
    `go build -o ${name} cmd/carto/main.go`,
    {
      image: `golang:${opts?.version || "1.22"}`,
      env: getGoBuildEnv(go),
    },
  );

  if (client && sha) {
    const { size } = await Deno.stat(name);
    await client.createCommitStatus("b1naryth1ef/carto", sha, {
      state: "success",
      description: formatBytes(size),
      context: `carto-${go.os}-${go.arch}`,
    });
  }

  if (release) {
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    await client.uploadReleaseAsset(
      release,
      name,
      await Deno.readFile(name),
    );
  }
}

export const github = webhook(async (event) => {
  if (event.push && event.push.head_commit) {
    for (const variant of matrix) {
      await spawnFn<typeof build>("build", {
        opts: { go: variant },
        sha: event.push.head_commit.id,
      }, { ref: event.push.head_commit.id });
    }
  } else if (event.create) {
    const client = await getClient();
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    if (event.create.ref_type === "tag" && event.create.ref.startsWith("v")) {
      const release = await client.createRelease("b1naryth1ef/carto", {
        tag: event.create.ref,
        name: event.create.ref,
        draft: true,
      });

      for (const variant of matrix) {
        await spawnFn<typeof build>("build", {
          opts: { go: variant },
          release: release,
        }, { ref: event.create.ref });
      }
    }
  }
});
