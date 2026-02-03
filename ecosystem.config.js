module.exports = {
  apps: [
    {
      name: 'voice-resolver',
      script: './main',
      cwd: '/root/voice_resolver',
      interpreter: 'none',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '500M',
      env: {
        PORT: 5000,
        DB_HOST: '10.0.0.4',
        DB_PORT: 5432,
        DB_USER: 'xdialcore',
        DB_PASSWORD: 'xdialcore',
        DB_NAME: 'xdialcore',
      },
      error_file: '/root/voice_resolver/logs/error.log',
      out_file: '/root/voice_resolver/logs/out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      min_uptime: '10s',
      max_restarts: 10,
    },
  ],
};

