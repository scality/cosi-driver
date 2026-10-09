import * as path from 'node:path';
import { defineConfig } from '@rspress/core';
import sidebar from './sidebar.json' with { type: 'json' };

const repo = 'https://github.com/scality/cosi-driver';

// Links to files outside docs/ (cosi-examples/, the chart) point at the commit
// the docs were built from: the release tag, or main for the preview.
const gitRef = process.env.DOCS_GIT_REF || 'main';
type MdNode = { type: string; url?: string; children?: MdNode[] };
const pinRepoLinks = () => (tree: MdNode) => {
  const walk = (node: MdNode) => {
    if (node.type === 'link' && node.url) {
      node.url = node.url.replace(new RegExp(`^${repo}/(blob|tree)/main/`), `${repo}/$1/${gitRef}/`);
    }
    node.children?.forEach(walk);
  };
  walk(tree);
};

export default defineConfig({
  // Replaced at load time by rspress-portable-base, so one build works
  // under every version folder and the latest symlink.
  base: '/REPLACE_ME_PREFIX/',
  root: path.join(__dirname, '..', 'docs'),
  title: 'Scality COSI Driver',
  description: 'Documentation for the Scality COSI Driver',
  icon: '/scality-logo.png',
  logo: '/scality-logo.png',
  logoText: 'COSI Driver',
  markdown: {
    link: { checkDeadLinks: true },
    remarkPlugins: [pinRepoLinks],
  },
  themeConfig: {
    llmsUI: { viewOptions: false },
    sidebar: { '/': sidebar },
    socialLinks: [{ icon: 'github', mode: 'link', content: repo }],
    editLink: { docRepoBaseUrl: `${repo}/edit/main/docs/` },
    footer: {
      message: `Licensed under the <a href="${repo}/blob/main/LICENSE">Apache License 2.0</a>`,
    },
  },
  llms: true,
});
