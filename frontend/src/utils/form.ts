import type { FormInstance } from 'antd';
import { fieldErrorsOf } from './error';

/** Map backend validation errors onto matching inputs. Details keys are DTO json names, matching form fields (no camelCase layer). Returns true when at least one error lands, so the screen skips the toast. Warning: unmatched keys are silently swallowed by antd, so the form must declare every field the endpoint validates. */
export const applyApiFieldErrors = (form: FormInstance, error: unknown): boolean => {
  const details = fieldErrorsOf(error);
  if (!details) return false;

  form.setFields(
    Object.entries(details).map(([name, message]) => ({
      name,
      errors: [message],
    }))
  );
  return true;
};
