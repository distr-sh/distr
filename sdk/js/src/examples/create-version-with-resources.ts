import {DistrService} from '../client';
import {clientConfig} from './config';

const distr = new DistrService(clientConfig);
const appId = '<docker-application-id>';

const composeFile = `
services:
  my-postgres:
    image: 'postgres:17.2-alpine3.20'
    ports:
      - '5434:5432'
    environment:
      POSTGRES_USER: \${POSTGRES_USER}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
      POSTGRES_DB: \${POSTGRES_DB}
    volumes:
      - 'postgres-data:/var/lib/postgresql/data/'

volumes:
  postgres-data:
`;

await distr.createDockerApplicationVersion(appId, '17.2-alpine3.20+3', {
  composeFile,
  resources: [
    {
      name: 'Getting Started',
      content:
        '# Getting Started\n\nFollow these steps to set up your database:\n\n1. Configure environment variables\n2. Run the compose file\n3. Connect to the database on port 5434',
      visibleToCustomers: true,
    },
    {
      name: 'Internal Notes',
      content: '# Internal Notes\n\nThis version uses PostgreSQL 17.2 on Alpine 3.20.',
      visibleToCustomers: false,
    },
  ],
});
