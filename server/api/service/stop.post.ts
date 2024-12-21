import { stopService } from '../../server'

export default defineEventHandler(async (event) => {
    const { service } = await readBody(event)
    await stopService(service)
    return { success: true }
})
