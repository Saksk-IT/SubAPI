import { spawn } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const command = (name) => process.platform === 'win32' ? `${name}.cmd` : name
const processes = [
  spawn(command('pnpm'), ['run', 'dev'], {
    cwd: frontendDirectory,
    stdio: 'inherit',
  }),
  spawn(command('npm'), ['run', 'dev'], {
    cwd: resolve(frontendDirectory, 'image-playground'),
    stdio: 'inherit',
  }),
]

let shuttingDown = false

function stopAll(signal = 'SIGTERM') {
  if (shuttingDown) return
  shuttingDown = true
  for (const child of processes) {
    if (!child.killed) child.kill(signal)
  }
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    stopAll(signal)
    process.exitCode = signal === 'SIGINT' ? 130 : 143
  })
}

for (const child of processes) {
  child.on('error', (error) => {
    console.error(`Failed to start frontend development server: ${error.message}`)
    stopAll()
    process.exitCode = 1
  })
  child.on('exit', (code, signal) => {
    if (shuttingDown) return
    stopAll()
    process.exitCode = signal ? 1 : (code ?? 1)
  })
}
