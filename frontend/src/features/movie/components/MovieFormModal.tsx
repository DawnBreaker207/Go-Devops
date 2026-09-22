import { useState } from 'react';
import { App, Button, DatePicker, Form, Input, InputNumber, Modal, Select, Upload } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  MOVIE_AGE_RATINGS,
  POSTER_ACCEPT,
  POSTER_MAX_BYTES,
  type Movie,
  type MoviePayload,
  type MovieStatus,
} from '@/types';
import { mediaApi } from '@/api/media.api';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

interface FormValues extends Omit<MoviePayload, 'release_date'> {
  release_date: dayjs.Dayjs;
}

interface MovieFormModalProps {
  open: boolean;
  movie: Movie | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throw on failure so the modal stays open and surfaces it; resolve lets the page close it. */
  onSubmit: (payload: MoviePayload) => Promise<void>;
}

/** antd already draws validateFields errors itself; never toast them again. */
const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const MovieFormModal = ({
  open,
  movie,
  confirmLoading,
  onCancel,
  onSubmit,
}: MovieFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const [uploading, setUploading] = useState(false);

  /** Uploads and writes the returned URL straight into the field, so the operator
   *  never has to host the image anywhere else first. */
  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await mediaApi.uploadPoster(file);
      form.setFieldValue('poster_url', result.url);
      // validateFields on one field: the url rule must re-run now that the value
      // changed programmatically rather than by typing.
      await form.validateFields(['poster_url']);
      message.success(t('movie.posterUploaded'));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setUploading(false);
    }
  };

  // Seed via initialValues only; saving replaces the record so list every field.
  const initialValues: Partial<FormValues> = movie
    ? {
        title: movie.title,
        genre: movie.genre,
        duration: movie.duration,
        director: movie.director,
        description: movie.description,
        poster_url: movie.poster_url,
        trailer_url: movie.trailer_url,
        cast: movie.cast,
        age_rating: movie.age_rating,
        status: movie.status,
        release_date: dayjs(movie.release_date),
      }
    : { status: 'draft' as MovieStatus, duration: 90, age_rating: 'P' };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit({
        ...values,
        release_date: values.release_date.format('YYYY-MM-DD'),
      });
    } catch (error) {
      if (isFormValidationError(error)) return;
      // 400/40001 returns details {json tag: message}: attach straight onto the input instead of
      // a generic toast that never says which field is wrong.
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  return (
    <Modal
      open={open}
      title={movie ? t('movie.editTitle') : t('movie.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={640}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item name="title" label={t('movie.name')} rules={[required]}>
          <Input placeholder="Inception" maxLength={255} />
        </Form.Item>

        <Form.Item name="genre" label={t('movie.genre')} rules={[required]}>
          <Input placeholder="Sci-Fi" maxLength={100} />
        </Form.Item>

        <Form.Item name="duration" label={t('movie.duration')} rules={[required]}>
          <InputNumber
            min={1}
            max={600}
            addonAfter={t('common.minutes')}
            style={{ width: '100%' }}
          />
        </Form.Item>

        <Form.Item name="director" label={t('movie.director')} rules={[required]}>
          <Input placeholder="Christopher Nolan" maxLength={255} />
        </Form.Item>

        <Form.Item name="cast" label={t('movie.cast')}>
          <Input.TextArea rows={2} maxLength={2000} placeholder="Leonardo DiCaprio, ..." />
        </Form.Item>

        <Form.Item name="release_date" label={t('movie.releaseDate')} rules={[required]}>
          <DatePicker style={{ width: '100%' }} format="DD/MM/YYYY" />
        </Form.Item>

        <Form.Item name="age_rating" label={t('movie.ageRating')}>
          <Select options={MOVIE_AGE_RATINGS.map((value) => ({ value, label: value }))} />
        </Form.Item>

        <Form.Item name="status" label={t('movie.status')} rules={[required]}>
          <Select
            options={[
              { value: 'draft', label: t('movie.statusDraft') },
              { value: 'showing', label: t('movie.statusShowing') },
              { value: 'ended', label: t('movie.statusEnded') },
            ]}
          />
        </Form.Item>

        {/* Backend binds `url`: relative paths like /media/x.jpg are rejected with 400.
            The upload returns an ABSOLUTE url built from storage.public_base_url,
            so what it hands back is directly valid here. */}
        <Form.Item
          name="poster_url"
          label={t('movie.posterUrl')}
          extra={t('movie.posterUploadHint')}
          rules={[{ type: 'url', message: t('movie.urlInvalid') }]}
        >
          <Input
            placeholder="https://..."
            maxLength={512}
            addonAfter={
              <Upload
                accept={POSTER_ACCEPT.join(',')}
                showUploadList={false}
                // customRequest, not `action`: antd's own uploader would bypass
                // the shared client and send no Authorization header.
                beforeUpload={(file) => {
                  if (!POSTER_ACCEPT.includes(file.type as (typeof POSTER_ACCEPT)[number])) {
                    message.error(t('movie.posterTypeInvalid'));
                    return Upload.LIST_IGNORE;
                  }
                  if (file.size > POSTER_MAX_BYTES) {
                    message.error(t('movie.posterTooLarge'));
                    return Upload.LIST_IGNORE;
                  }
                  void handleUpload(file);
                  // Always false: the upload is done by handleUpload above.
                  return false;
                }}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<UploadOutlined />}
                  loading={uploading}
                  aria-label={t('movie.posterUpload')}
                />
              </Upload>
            }
          />
        </Form.Item>

        <Form.Item
          name="trailer_url"
          label={t('movie.trailerUrl')}
          rules={[{ type: 'url', message: t('movie.urlInvalid') }]}
        >
          <Input placeholder="https://www.youtube.com/watch?v=..." maxLength={512} />
        </Form.Item>

        <Form.Item name="description" label={t('movie.description')}>
          <Input.TextArea rows={3} maxLength={5000} showCount />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default MovieFormModal;
