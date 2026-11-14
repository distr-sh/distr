import {AbstractControl, FormControl, FormGroup, ValidationErrors, ValidatorFn} from '@angular/forms';
import {DeploymentTargetSettings, DeploymentTargetToleration} from '@distr-sh/distr-sdk';

const TOLERATION_EFFECTS = ['NoSchedule', 'PreferNoSchedule', 'NoExecute'] as const;

function lines(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line !== '');
}

/**
 * Parses one `key=value` pair per line. Throws on a line without a key.
 */
export function parseKeyValueLines(text: string): Record<string, string> | undefined {
  const result: Record<string, string> = {};
  for (const line of lines(text)) {
    const separator = line.indexOf('=');
    const key = separator < 0 ? line : line.slice(0, separator).trim();
    if (separator < 0 || key === '') {
      throw new Error(`"${line}" is not a key=value pair`);
    }
    result[key] = line.slice(separator + 1).trim();
  }
  return Object.keys(result).length > 0 ? result : undefined;
}

export function formatKeyValueLines(value: Record<string, string> | undefined): string {
  return Object.entries(value ?? {})
    .map(([key, val]) => `${key}=${val}`)
    .join('\n');
}

/**
 * Parses one toleration per line in the taint syntax of kubectl, `key[=value][:effect]`. A key without a value
 * tolerates the taint whatever its value is, and an empty key with no value tolerates every taint.
 */
export function parseTolerationLines(text: string): DeploymentTargetToleration[] | undefined {
  const result = lines(text).map((line): DeploymentTargetToleration => {
    let rest = line;
    let effect: DeploymentTargetToleration['effect'];
    const effectSeparator = rest.lastIndexOf(':');
    if (effectSeparator >= 0) {
      const candidate = rest.slice(effectSeparator + 1).trim();
      if (!(TOLERATION_EFFECTS as readonly string[]).includes(candidate)) {
        throw new Error(`"${candidate}" in "${line}" is not one of ${TOLERATION_EFFECTS.join(', ')}`);
      }
      effect = candidate as DeploymentTargetToleration['effect'];
      rest = rest.slice(0, effectSeparator).trim();
    }
    const valueSeparator = rest.indexOf('=');
    if (valueSeparator < 0) {
      return {key: rest || undefined, operator: 'Exists', effect};
    }
    const key = rest.slice(0, valueSeparator).trim();
    if (key === '') {
      throw new Error(`"${line}" has a value but no key`);
    }
    return {key, operator: 'Equal', value: rest.slice(valueSeparator + 1).trim(), effect};
  });
  return result.length > 0 ? result : undefined;
}

export function formatTolerationLines(value: DeploymentTargetToleration[] | undefined): string {
  return (value ?? [])
    .map((t) => {
      const keyValue = t.operator === 'Exists' ? (t.key ?? '') : `${t.key ?? ''}=${t.value ?? ''}`;
      return t.effect ? `${keyValue}:${t.effect}` : keyValue;
    })
    .join('\n');
}

function parsableValidator(parse: (text: string) => unknown): ValidatorFn {
  return (control: AbstractControl<string>): ValidationErrors | null => {
    try {
      parse(control.value ?? '');
      return null;
    } catch (e) {
      return {parse: (e as Error).message};
    }
  };
}

type PodScheduling = Pick<DeploymentTargetSettings, 'podAnnotations' | 'nodeSelector' | 'tolerations'>;

export function podSchedulingFormGroup() {
  return new FormGroup({
    podAnnotations: new FormControl('', {nonNullable: true, validators: parsableValidator(parseKeyValueLines)}),
    nodeSelector: new FormControl('', {nonNullable: true, validators: parsableValidator(parseKeyValueLines)}),
    tolerations: new FormControl('', {nonNullable: true, validators: parsableValidator(parseTolerationLines)}),
  });
}

export type PodSchedulingFormGroup = ReturnType<typeof podSchedulingFormGroup>;

export function podSchedulingFromForm(group: PodSchedulingFormGroup): PodScheduling {
  const value = group.getRawValue();
  return {
    podAnnotations: parseKeyValueLines(value.podAnnotations),
    nodeSelector: parseKeyValueLines(value.nodeSelector),
    tolerations: parseTolerationLines(value.tolerations),
  };
}

export function podSchedulingToForm(value: PodScheduling): PodSchedulingFormGroup['value'] {
  return {
    podAnnotations: formatKeyValueLines(value.podAnnotations),
    nodeSelector: formatKeyValueLines(value.nodeSelector),
    tolerations: formatTolerationLines(value.tolerations),
  };
}

export function hasPodScheduling(value: PodScheduling): boolean {
  return (
    Object.keys(value.podAnnotations ?? {}).length > 0 ||
    Object.keys(value.nodeSelector ?? {}).length > 0 ||
    (value.tolerations?.length ?? 0) > 0
  );
}
