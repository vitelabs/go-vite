const container = require('markdown-it-container')

module.exports = (opts, ctx) => {
  return {
    extendMarkdown: md => {
      md.use(container, 'demo', {
        render: function (tokens, idx) {
          const token = tokens[idx];
          if (token.type === 'container_demo_open') {
            return '<code-group>'
          } else {
            return '</code-group>'
          }
        }
      })
      md.use(container, 'tabs', {
        render: function (tokens, idx) {
          const token = tokens[idx];
          if (token.type === 'container_tabs_open') {
            return '<code-group>'
          } else {
            return '</code-group>'
          }
        }
      })
      md.use(require('./markdown-it-code-block.js'))
    }
  }
}
