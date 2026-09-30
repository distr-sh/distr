import {BaseModel} from './base';

export interface ControllerVersion extends BaseModel {
  name: string;
}

/** @deprecated Use {@link ControllerVersion} instead. */
export type AgentVersion = ControllerVersion;
