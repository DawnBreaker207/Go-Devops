import type { FormInstance } from 'antd';
import { fieldErrorsOf } from './error';

/**
 * Gan loi validate cua backend vao dung o nhap.
 *
 * Key trong details la ten json tag cua DTO, tuc la trung ten field cua form neu
 * form dat ten theo wire (repo nay khong co lop map camelCase). Tra ve true khi
 * da gan duoc it nhat mot loi - luc do man hinh KHONG toast nua, loi da nam canh
 * o nhap roi.
 *
 * Luu y: mot key khong co o nhap tuong ung thi antd van nhan nhung khong ve ra
 * dau ca. Form nao submit len endpoint nao thi phai co du field endpoint do
 * validate, neu khong loi se im lang.
 */
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
