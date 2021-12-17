const fs = require("fs");

function replaceUnsafeChar(ch) {
  return HTML_REPLACEMENTS[ch];
}

var HTML_REPLACEMENTS = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
};

function escapeHtml(str) {
  if (/[&<>"]/.test(str)) {
    return str.replace(/[&<>"]/g, replaceUnsafeChar);
  }
  return str;
}

module.exports = (md) => {
  md.renderer.rules.fence = (...args) => {
    const [tokens, idx, options] = args;
    const token = tokens[idx];
    if (fs.existsSync(token.src)) {
      token.content = fs.readFileSync(token.src, "utf8");
    }
    const base64 = Buffer.from(escapeHtml(token.content)).toString("base64");
    const [lang, ext] = token.info.split(' ');
    if (ext && ext.startsWith('tab:')) {
      const title= ext.replace(/tab:/g, '');
      return `
      <code-block title="${title}">
      <tm-code-block class="codeblock" language="${lang}" base64="${base64}"></tm-code-block>
      </code-block>
      `;
    } else {
      return `
      <tm-code-block class="codeblock" language="${lang}" base64="${base64}"></tm-code-block>
      `;
    }
  };
};
