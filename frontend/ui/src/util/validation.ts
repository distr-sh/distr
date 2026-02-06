import {AbstractControl, FormRecord, ValidationErrors, ValidatorFn} from '@angular/forms';

export const KUBERNETES_RESOURCE_MAX_LENGTH = 253;
export const KUBERNETES_RESOURCE_NAME_REGEX = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
export const HELM_RELEASE_NAME_REGEX = /^[a-z0-9]([-a-z0-9]*)?[a-z0-9]$/;
export const HELM_RELEASE_NAME_MAX_LENGTH = 53;

/**
 * The serialization format is:
 *
 * ```
 * <quantity>        ::= <signedNumber><suffix>
 *
 * (Note that <suffix> may be empty, from the "" case in <decimalSI>.)
 *
 * <digit>           ::= 0 | 1 | ... | 9
 * <digits>          ::= <digit> | <digit><digits>
 * <number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits>
 * <sign>            ::= "+" | "-"
 * <signedNumber>    ::= <number> | <sign><number>
 * <suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI>
 * <binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei
 *
 * (International System of units; See: http://physics.nist.gov/cuu/Units/binary.html)
 *
 * <decimalSI>       ::= m | "" | k | M | G | T | P | E
 * (Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)
 *
 * <decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>
 * ```
 *
 * Source: https://github.com/kubernetes/apimachinery/blob/v0.35.0/pkg/api/resource/quantity.go#L103
 */
export const RESOURCE_QUANTITY_REGEX =
  /^(\d+|\d+\.\d+|\d+\.|\.\d+)(m|k|M|G|T|P|E|Ki|Mi|Gi|Ti|Pi|Ei|((e|E)(\d+|\d+\.\d+|\d+\.|\.\d+)))?$/;

export function validateRecordAtLeast(minRequiredCount: number, evalFunc: (v: any) => unknown = (v) => v): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (control instanceof FormRecord) {
      const truthyValuesCount = Object.values(control.value).filter(evalFunc).length;
      if (truthyValuesCount < minRequiredCount) {
        return {validateRecordAtLeast: true};
      }
    }

    return null;
  };
}
