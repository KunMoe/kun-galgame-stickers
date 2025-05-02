import path from 'path'
import { promises as fs } from 'fs'
import { fileURLToPath } from 'node:url'
import mongoose from 'mongoose'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const mongoURI = 'mongodb://127.0.0.1:27017/kungalgame_sticker'
mongoose.connect(mongoURI, { useNewUrlParser: true, useUnifiedTopology: true })

const KUNStickerSchema = new mongoose.Schema(
  {
    sid: { type: Number, default: 0 },
    pid: { type: Number, default: 0 },
    src: { type: String, default: '' },

    game: {
      'en-us': { type: String, default: '' },
      'ja-jp': { type: String, default: '' },
      'zh-cn': { type: String, default: '' },
      'zh-tw': { type: String, default: '' }
    },
    loli: {
      'en-us': { type: String, default: '' },
      'ja-jp': { type: String, default: '' },
      'zh-cn': { type: String, default: '' },
      'zh-tw': { type: String, default: '' }
    },
    vndb: { type: Number, default: 0 },
    describe: { type: String, default: '' }
  },
  { timestamps: { createdAt: 'created', updatedAt: 'updated' } }
)
const KUNStickerModel = mongoose.model('stickers', KUNStickerSchema)

// Function to read and parse a JSON file
async function readAndParseJSON(filePath) {
  const data = await fs.readFile(filePath, 'utf8')
  return JSON.parse(data)
}

async function migrateData() {
  const gameFolderPathEn = path.join(__dirname, 'en/game')
  const lassFolderPathEn = path.join(__dirname, 'en/lass')

  const gameFolderPathZh = path.join(__dirname, 'zh/game')
  const lassFolderPathZh = path.join(__dirname, 'zh/lass')

  for (let i = 1; i <= 7; i++) {
    const gameDataEn = await readAndParseJSON(path.join(gameFolderPathEn, `game${i}.json`))
    const lassDataEn = await readAndParseJSON(path.join(lassFolderPathEn, `lass${i}.json`))
    const gameDataZh = await readAndParseJSON(path.join(gameFolderPathZh, `game${i}.json`))
    const lassDataZh = await readAndParseJSON(path.join(lassFolderPathZh, `lass${i}.json`))

    for (const pid in gameDataEn) {
      const sticker = new KUNStickerModel({
        sid: i,
        pid: parseInt(pid, 10),
        game: {
          'en-us': gameDataEn[pid],
          'ja-jp': '',
          'zh-cn': gameDataZh[pid],
          'zh-tw': ''
        },
        loli: {
          'en-us': lassDataEn[pid],
          'ja-jp': '',
          'zh-cn': lassDataZh[pid],
          'zh-tw': ''
        }
      })
      await sticker.save()
    }
  }
}

migrateData().then(() => console.log('Migration completed.'))
