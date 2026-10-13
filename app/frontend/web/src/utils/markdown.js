import { marked } from 'marked';
import hljs from 'highlight.js/lib/core';
import go from 'highlight.js/lib/languages/go';
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import python from 'highlight.js/lib/languages/python';
import bash from 'highlight.js/lib/languages/bash';
import json from 'highlight.js/lib/languages/json';
import yaml from 'highlight.js/lib/languages/yaml';
import sql from 'highlight.js/lib/languages/sql';
import dockerfile from 'highlight.js/lib/languages/dockerfile';
import markdown from 'highlight.js/lib/languages/markdown';
import xml from 'highlight.js/lib/languages/xml';

hljs.registerLanguage('go', go);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('python', python);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('json', json);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('dockerfile', dockerfile);
hljs.registerLanguage('markdown', markdown);
hljs.registerLanguage('xml', xml);

marked.setOptions({
  breaks: true,
  gfm: true,
});

function normalizeMarkdown(text) {
  let result = text;

  // Fix headings without space: "##text" → "## text"
  result = result.replace(/^(#{1,6})([^ \n#])/gm, '$1 $2');

  // Fix merged horizontal rule + heading: "---##text" → "---\n\n## text"
  result = result.replace(/---(#{1,6})/g, '---\n\n$1');

  return result;
}

export function renderMarkdown(text) {
  if (!text) return '';
  try {
    const normalized = normalizeMarkdown(text);
    const html = marked.parse(normalized);
    if (typeof html !== 'string' || html === text) {
      return text;
    }
    return html;
  } catch (e) {
    console.error('Markdown parse error:', e);
    try {
      highlightCodeBlocks();
    } catch (_) {}
    return text;
  }
}

function highlightCodeBlocks() {
  document.querySelectorAll('.markdown-content pre code').forEach((block) => {
    const classes = block.className.split(' ');
    const langClass = classes.find((c) => c.startsWith('language-'));
    if (langClass) {
      const lang = langClass.replace('language-', '');
      if (lang && hljs.getLanguage(lang)) {
        const result = hljs.highlight(block.textContent, { language: lang });
        block.innerHTML = result.value;
        block.classList.add('hljs');
      }
    } else {
      const result = hljs.highlightAuto(block.textContent);
      block.innerHTML = result.value;
      block.classList.add('hljs');
    }
  });
}

export function formatTimestamp(timestamp) {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now - date;
  
  if (diff < 60000) return 'Just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  
  return date.toLocaleDateString();
}