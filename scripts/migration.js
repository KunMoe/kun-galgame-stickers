import { PrismaClient } from '@prisma/client'
import path from 'path'
import { promises as fs } from 'fs'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const prisma = new PrismaClient()

const readAndParseJSON = async (filePath) => {
  const data = await fs.readFile(filePath, 'utf8')
  return JSON.parse(data)
}

const migrateData = async () => {
  try {
    const dataToCreate = []

    const gameFolderPathEn = path.join(__dirname, 'en/game')
    const lassFolderPathEn = path.join(__dirname, 'en/lass')
    const gameFolderPathZh = path.join(__dirname, 'zh/game')
    const lassFolderPathZh = path.join(__dirname, 'zh/lass')

    for (let i = 1; i <= 7; i++) {
      console.log(`Processing set ${i}...`)
      const gameDataEn = await readAndParseJSON(path.join(gameFolderPathEn, `game${i}.json`))
      const lassDataEn = await readAndParseJSON(path.join(lassFolderPathEn, `lass${i}.json`))
      const gameDataZh = await readAndParseJSON(path.join(gameFolderPathZh, `game${i}.json`))
      const lassDataZh = await readAndParseJSON(path.join(lassFolderPathZh, `lass${i}.json`))

      for (const pid in gameDataEn) {
        const stickerRecord = {
          sid: i,
          pid: parseInt(pid, 10),

          game: {
            'en-us': gameDataEn[pid] || '',
            'ja-jp': '',
            'zh-cn': gameDataZh[pid] || '',
            'zh-tw': ''
          },
          loli: {
            'en-us': lassDataEn[pid] || '',
            'ja-jp': '',
            'zh-cn': lassDataZh[pid] || '',
            'zh-tw': ''
          }
        }
        dataToCreate.push(stickerRecord)
      }
    }

    const result = await prisma.sticker.createMany({
      data: dataToCreate,
      skipDuplicates: true
    })

    console.log(`Migration completed. Inserted ${result.count} new records.`)
  } catch (error) {
    console.error('An error occurred during migration:', error)
  } finally {
    await prisma.$disconnect()
  }
}

migrateData()
