import { App, DatePicker, Form, Modal, Select } from 'antd';
import type dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { Showtime, ShowtimePayload, ShowtimeStatus } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { fromApiInstant, toApiInstant } from '@/utils/format';
import { useHallOptions, useMovieOptions } from '../hooks/useShowtimes';

interface FormValues {
  movie_id: string;
  hall_id: string;
  start_at: dayjs.Dayjs;
  status: ShowtimeStatus;
}

interface ShowtimeFormModalProps {
  open: boolean;
  showtime: Showtime | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Nem loi ra thi modal giu nguyen va tu hien thi. */
  onSubmit: (payload: ShowtimePayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const ShowtimeFormModal = ({
  open,
  showtime,
  confirmLoading,
  onCancel,
  onSubmit,
}: ShowtimeFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const movies = useMovieOptions();
  const halls = useHallOptions();

  // Nap bang initialValues chu khong bang setFieldsValue trong useEffect - xem
  // .claude/rules/pages-components.md, StrictMode se xoa mat gia tri.
  const initialValues: Partial<FormValues> = showtime
    ? {
        movie_id: showtime.movie_id,
        hall_id: showtime.hall_id,
        start_at: fromApiInstant(showtime.start_at),
        status: showtime.status,
      }
    : { status: 'open' };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // PUT doi du movie_id + hall_id + start_at ke ca khi chi doi trang thai.
      await onSubmit({
        movie_id: values.movie_id,
        hall_id: values.hall_id,
        start_at: toApiInstant(values.start_at),
        status: values.status,
      });
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  return (
    <Modal
      open={open}
      title={showtime ? t('showtime.editTitle') : t('showtime.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={560}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item name="movie_id" label={t('showtime.movie')} rules={[required]}>
          <Select
            showSearch
            optionFilterProp="label"
            loading={movies.isFetching}
            placeholder={t('showtime.moviePlaceholder')}
            options={(movies.data?.items ?? []).map((m) => ({ value: m.id, label: m.title }))}
          />
        </Form.Item>

        <Form.Item name="hall_id" label={t('showtime.hall')} rules={[required]}>
          <Select
            showSearch
            optionFilterProp="label"
            loading={halls.isFetching}
            placeholder={t('showtime.hallPlaceholder')}
            options={(halls.data?.items ?? []).map((h) => ({
              value: h.id,
              label: h.active ? h.name : `${h.name} (${t('showtime.hallInactive')})`,
              disabled: !h.active,
            }))}
          />
        </Form.Item>

        {/* Gio nhap la gio RAP. end_at do backend suy ra tu thoi luong phim. */}
        <Form.Item
          name="start_at"
          label={t('showtime.startAt')}
          extra={t('showtime.startAtHint')}
          rules={[required]}
        >
          <DatePicker
            showTime={{ format: 'HH:mm', minuteStep: 5 }}
            format="DD/MM/YYYY HH:mm"
            style={{ width: '100%' }}
          />
        </Form.Item>

        {/* POST bo qua status (luon tao 'open'), nen chi cho doi khi sua. */}
        {showtime ? (
          <Form.Item name="status" label={t('showtime.status')} rules={[required]}>
            <Select
              options={[
                { value: 'open', label: t('showtime.statusOpen') },
                { value: 'closed', label: t('showtime.statusClosed') },
              ]}
            />
          </Form.Item>
        ) : null}
      </Form>
    </Modal>
  );
};

export default ShowtimeFormModal;
