const path = require('path')

module.exports = {
  apps: [
    {
      name: 'kun-visual-novel-sticker',
      port: 9420,
      cwd: path.join(__dirname),
      instances: 1,
      autorestart: true,
      watch: false,
      max_memory_restart: '1G',
      script: './kun-love-ren/index.js'
    }
  ]
}
