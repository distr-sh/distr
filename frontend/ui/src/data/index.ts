import * as agentChangelog from './agent-changelog.json';
import * as buildConfig from './version.json';

export interface AgentChangelogChange {
  scope: string;
  description: string;
  commit: string;
  pr?: number;
}

export interface AgentChangelogSection {
  section: string;
  changes: AgentChangelogChange[];
}

export interface AgentChangelogRelease {
  version: string;
  sections: AgentChangelogSection[];
}

export interface AgentChangelog {
  releases: AgentChangelogRelease[];
}

const typedAgentChangelog = agentChangelog as AgentChangelog;

export {typedAgentChangelog as agentChangelog, buildConfig};
