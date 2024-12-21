import { startService } from '../../server'

export default defineEventHandler(async (event) => {
    const { service } = await readBody(event)
    await startService(service)
    return { success: true }
})
