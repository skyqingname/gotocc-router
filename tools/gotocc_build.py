#!/usr/bin/env python3
"""Build the frontend and embedded server with the same commands locally and in CI."""
import argparse
import json
from pathlib import Path
import shlex
import subprocess


ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--ldflags', default='')
    parser.add_argument('--plan', action='store_true')
    args = parser.parse_args()
    config = json.loads((ROOT / 'gotocc-build.json').read_text())
    output = args.output.resolve()
    steps = [
        ['pnpm', '--dir', 'frontend', 'install', *config['frontend_install_args']],
        ['pnpm', '--dir', 'frontend', 'exec', 'node', f'--max-old-space-size={config["node_heap_mb"]}',
         'node_modules/vue-tsc/bin/vue-tsc.js', '-b'],
        ['pnpm', '--dir', 'frontend', 'exec', 'node', f'--max-old-space-size={config["node_heap_mb"]}',
         'node_modules/vite/bin/vite.js', 'build'],
        ['go', '-C', 'backend', 'build', '-tags=' + ','.join(config['go_build_tags']),
         *(['-trimpath'] if config['go_trimpath'] else []), '-ldflags=' + args.ldflags,
         '-o', str(output), './cmd/server'],
    ]
    if not args.plan:
        output.parent.mkdir(parents=True, exist_ok=True)
    for step in steps:
        print(shlex.join(step), flush=True)
        if not args.plan:
            subprocess.run(step, cwd=ROOT, check=True)


if __name__ == '__main__':
    main()
