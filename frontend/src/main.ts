import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import './style.css'
import App from './App.vue'

// Import Wails runtime
import { EventsOn } from '../wailsjs/runtime/runtime.js'

// Create router
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: App }
  ]
})

// Create app
const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Listen for service events from Go backend
EventsOn('serviceEvent', (event) => {
  // Handle service events
  console.log('Service event:', event)
})

app.mount('#app')