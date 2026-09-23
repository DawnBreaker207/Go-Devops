import { useEffect } from 'react';
import { DatePicker, Form, Input, InputNumber, Modal, Select } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { Movie, MoviePayload, MovieStatus } from '@/types';

interface FormValues extends Omit<MoviePayload, 'release_date'> {
  release_date: dayjs.Dayjs;
}

interface MovieFormModalProps {
  open: boolean;
  movie: Movie | null;
  confirmLoading: boolean;
  onCancel: () => void;
  onSubmit: (payload: MoviePayload) => void;
}

export const MovieFormModal = ({
  open,
  movie,
  confirmLoading,
  onCancel,
  onSubmit,
}: MovieFormModalProps) => {
  const { t } = useTranslation();
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (!open) return;
    if (movie) {
      form.setFieldsValue({
        title: movie.title,
        genre: movie.genre,
        duration: movie.duration,
        director: movie.director,
        description: movie.description,
        poster_url: movie.poster_url,
        status: movie.status,
        release_date: dayjs(movie.release_date),
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ status: 'draft' as MovieStatus, duration: 90 });
    }
  }, [open, movie, form]);

  const handleOk = async () => {
    const values = await form.validateFields();
    onSubmit({
      ...values,
      release_date: values.release_date.format('YYYY-MM-DD'),
    });
  };

  return (
    <Modal
      open={open}
      title={movie ? t('movie.editTitle') : t('movie.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnClose
      width={640}
    >
      <Form<FormValues> form={form} layout="vertical" preserve={false}>
        <Form.Item name="title" label={t('movie.name')} rules={[{ required: true }]}>
          <Input placeholder="Inception" />
        </Form.Item>

        <Form.Item name="genre" label={t('movie.genre')} rules={[{ required: true }]}>
          <Input placeholder="Sci-Fi" />
        </Form.Item>

        <Form.Item
          name="duration"
          label={`${t('movie.duration')} (phút)`}
          rules={[{ required: true }]}
        >
          <InputNumber min={1} max={600} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="director" label={t('movie.director')} rules={[{ required: true }]}>
          <Input placeholder="Christopher Nolan" />
        </Form.Item>

        <Form.Item name="release_date" label={t('movie.releaseDate')} rules={[{ required: true }]}>
          <DatePicker style={{ width: '100%' }} format="DD/MM/YYYY" />
        </Form.Item>

        <Form.Item name="status" label={t('movie.status')} rules={[{ required: true }]}>
          <Select
            options={[
              { value: 'draft', label: t('movie.statusDraft') },
              { value: 'showing', label: t('movie.statusShowing') },
              { value: 'ended', label: t('movie.statusEnded') },
            ]}
          />
        </Form.Item>

        <Form.Item name="poster_url" label={t('movie.posterUrl')}>
          <Input placeholder="https://..." />
        </Form.Item>

        <Form.Item name="description" label={t('movie.description')}>
          <Input.TextArea rows={3} maxLength={1000} showCount />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default MovieFormModal;
