import { App, DatePicker, Form, Modal, Select, Typography } from 'antd';
import type dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { Showtime, ShowtimePayload } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { fromApiInstant, toApiInstant } from '@/utils/format';
import { useHallOptions, useMovieOptions } from '../hooks/useShowtimes';

interface FormValues {
  movie_id: string;
  hall_id: string;
  start_at: dayjs.Dayjs;
  // Cancelling goes through its own endpoint; this PUT only takes open/closed.
  status: 'open' | 'closed';
}

interface ShowtimeFormModalProps {
  open: boolean;
  showtime: Showtime | null;
  confirmLoading: boolean;
  onCancel: () => void;
  // Throw so the modal stays open binding errors onto inputs.
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

  // end_at is derived by the backend from runtime; shown here as preview only.
  const watchedMovieId = Form.useWatch('movie_id', form);
  const watchedStartAt = Form.useWatch('start_at', form);
  const selectedMovie = movies.data?.items.find((m) => m.id === watchedMovieId);
  const estimatedEndAt =
    selectedMovie && watchedStartAt ? watchedStartAt.add(selectedMovie.duration, 'minute') : null;

  // Seed via initialValues, never setFieldsValue in an effect: StrictMode wipes it.
  const initialValues: Partial<FormValues> = showtime
    ? {
        movie_id: showtime.movie_id,
        hall_id: showtime.hall_id,
        start_at: fromApiInstant(showtime.start_at),
        // Cancelled is terminal and unpickable; the form disables itself too.
        status: showtime.status === 'cancelled' ? 'closed' : showtime.status,
      }
    : { status: 'open' };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // PUT wants full movie+hall+start_at even for a status-only change.
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
            options={(movies.data?.items ?? []).map((m) => ({
              value: m.id,
              // The backend only takes showing movies, so lock it in the dropdown instead of failing on Save.
              label:
                m.status === 'showing'
                  ? m.title
                  : `${m.title} (${t(`movie.status${m.status.charAt(0).toUpperCase()}${m.status.slice(1)}`)})`,
              disabled: m.status !== 'showing',
            }))}
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

        {/* Gio nhap la gio rap, end_at backend tu suy tu runtime, duoi chi la uoc tinh. */}
        <Form.Item
          name="start_at"
          label={t('showtime.startAt')}
          extra={
            estimatedEndAt ? (
              <Typography.Text type="secondary">
                {t('showtime.estimatedEndAt', { time: estimatedEndAt.format('HH:mm DD/MM') })}
              </Typography.Text>
            ) : (
              t('showtime.startAtHint')
            )
          }
          rules={[required]}
        >
          <DatePicker
            showTime={{ format: 'HH:mm', minuteStep: 5 }}
            format="DD/MM/YYYY HH:mm"
            style={{ width: '100%' }}
          />
        </Form.Item>

        {/* POST luon tao open nen chi hien status khi sua. */}
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
