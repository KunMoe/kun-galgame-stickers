import { PrismaPg } from '@prisma/adapter-pg'
import { PrismaClient } from '@prisma/client'
import { dev } from '$app/environment'
import { env } from '$env/dynamic/private'

const createPrismaClient = () =>
  new PrismaClient({
    adapter: new PrismaPg({ connectionString: env.KUN_DATABASE_URL })
  })

const globalForPrisma = globalThis as unknown as { prisma?: PrismaClient }

export const prisma = globalForPrisma.prisma ?? createPrismaClient()

if (dev) {
  globalForPrisma.prisma = prisma
}
