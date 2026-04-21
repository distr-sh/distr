import {Injectable} from '@angular/core';
import {agentChangelog} from '../../buildconfig';

export interface AgentChangelogEntry {
  scope: string;
  description: string;
  commit: string;
  pr?: number;
}

export interface AgentChangelogSection {
  section: string;
  changes: AgentChangelogEntry[];
}

export interface AgentChangelogRelease {
  version: string;
  sections: AgentChangelogSection[];
}

export interface AgentChangelog {
  releases: AgentChangelogRelease[];
}

@Injectable({providedIn: 'root'})
export class AgentChangelogService {
  private readonly changelog: AgentChangelog = agentChangelog;

  public get(): AgentChangelog {
    return this.changelog;
  }
}
