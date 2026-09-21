import { describe, expect, it } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import { Form, Input } from 'antd';
import type { FormInstance } from 'antd';
import { applyApiFieldErrors } from '../form';
import { errorMessage, fieldErrorsOf, isApiError, VALIDATION_ERROR_CODE } from '../error';

/** Real Form asserting errors RENDER next to inputs, not just sit in the store. */
const renderForm = () => {
  let form!: FormInstance;
  const Harness = () => {
    [form] = Form.useForm();
    return (
      <Form form={form} layout="vertical">
        <Form.Item name="title" label="Ten phim">
          <Input />
        </Form.Item>
      </Form>
    );
  };
  render(<Harness />);
  return form;
};

const validationError = (details: Record<string, string>) => ({
  code: VALIDATION_ERROR_CODE,
  message: 'validation failed',
  details,
});

describe('applyApiFieldErrors', () => {
  it('gan details cua 40001 vao dung o nhap', async () => {
    const form = renderForm();

    let applied = false;
    // setFields runs outside React render, so act() to flush errors to paint.
    act(() => {
      applied = applyApiFieldErrors(form, validationError({ title: 'must not be blank' }));
    });

    expect(applied).toBe(true);
    expect(form.getFieldError('title')).toEqual(['must not be blank']);
    // antd paints error rows via CSSMotion a tick later; wait for it.
    expect(await screen.findByText('must not be blank')).toBeInTheDocument();
  });

  it('bo qua loi khong phai 40001, de man hinh tu toast', () => {
    const form = renderForm();

    expect(applyApiFieldErrors(form, { code: 40900, message: 'conflict' })).toBe(false);
    expect(applyApiFieldErrors(form, new Error('network'))).toBe(false);
    expect(applyApiFieldErrors(form, validationError({}))).toBe(false);
  });
});

describe('error helpers', () => {
  it('nhan dien ApiError', () => {
    expect(isApiError({ code: 0, message: 'success' })).toBe(true);
    expect(isApiError(new Error('boom'))).toBe(false);
    expect(isApiError(null)).toBe(false);
  });

  it('errorMessage tra fallback khi loi khong den tu API', () => {
    expect(errorMessage({ code: 40400, message: 'not found' }, 'fallback')).toBe('not found');
    expect(errorMessage(new Error('boom'), 'fallback')).toBe('fallback');
  });

  it('details chi duoc doc khi code dung la 40001', () => {
    expect(fieldErrorsOf(validationError({ email: 'invalid' }))).toEqual({ email: 'invalid' });
    expect(
      fieldErrorsOf({ code: 40000, message: 'bad', details: { email: 'invalid' } })
    ).toBeUndefined();
  });
});
