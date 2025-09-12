import { execSync } from 'child_process'
import { config } from 'dotenv'
import { fileURLToPath } from 'url'
import { dirname } from 'path'
import * as fs from 'fs'
import * as path from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const envPath = path.resolve(__dirname, '..', '.env')
if (!fs.existsSync(envPath)) {
  console.error('.env file not found in the project root.')
  process.exit(1)
}

config({ path: envPath })

try {
  console.log('Executing the commands...')

  execSync('git pull && pnpm prisma:push && pnpm build && pnpm stop && pnpm start', {
    stdio: 'inherit'
  })
} catch (error) {
  console.error('Invalid environment variables', error)
  process.exit(1)
}
