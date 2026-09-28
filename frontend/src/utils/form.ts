import type { FormInstance } from 'antd';
import { fieldErrorsOf } from './error';

// snake_case both ways: details keys match field names, no conversion.
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
