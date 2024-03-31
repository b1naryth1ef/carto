import { spawn } from "@maf/core.ts";
import { run } from "@maf/docker/mod.ts";
import { getGoBuildEnv, GOARCH, GoBuild, GOOS } from "@maf/lang/go.ts";
import { getClient, Release, webhook } from "@maf/service/github.ts";
import { toReadableStream } from "@std/io/mod.ts";

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
  await Promise.all(matrix.map((it) => build({ go: it })));
}

export async function build(
  opts?: { go: GoBuild; version?: string },
  release?: Release,
) {
  const go = opts?.go || { os: GOOS.linux, arch: GOARCH.amd64 };
  let name = `carto-${go.os}.${go.arch}`;
  if (go.os === "windows") {
    name = name + ".exe";
  }

  await run(
    `go build -o ${name} cmd/carto/main.go`,
    {
      image: `golang:${opts?.version || "1.22"}`,
      env: getGoBuildEnv(go),
    },
  );

  if (release) {
    const client = await getClient();
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    using file = await Deno.open(name, { read: true });

    await client.uploadReleaseAsset(
      release,
      name,
      "application/octet-stream",
      toReadableStream(file),
    );
  }
}

export const github = webhook(async (event) => {
  if (event.push) {
    for (const build of matrix) {
      await spawn("build", {
        opts: { go: build },
      }, { ref: event.push.head_commit.id });
    }
  } else if (event.create) {
    const client = await getClient();
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    if (event.create.ref_type === "tag") {
      const release = await client.createRelease("b1naryth1ef/carto", {
        tag: event.create.ref,
        name: event.create.ref,
        draft: true,
      });

      for (const build of matrix) {
        await spawn("build", {
          opts: { go: build },
          release: release,
        }, { ref: event.create.ref });
      }
    }
  } else {
    console.log(event);
  }
});
