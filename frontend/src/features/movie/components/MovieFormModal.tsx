import { App, DatePicker, Form, Input, InputNumber, Modal, Select } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { MOVIE_AGE_RATINGS, type Movie, type MoviePayload, type MovieStatus } from '@/types';
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
  /** Nem loi ra thi modal giu nguyen va tu hien thi; resolve thi trang dong modal. */
  onSubmit: (payload: MoviePayload) => Promise<void>;
}

/** antd tu ve loi cua validateFields roi, khong toast them lan nua. */
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

  // Nap gia tri qua initialValues chu KHONG qua setFieldsValue trong useEffect.
  // Duoi StrictMode, React chay effect -> cleanup -> effect lai; cleanup cua Field
  // voi preserve={false} xoa gia tri khoi store va chay sau, nen thu tu cu de lai
  // mot form rong. destroyOnHidden dung Form moi cho moi lan mo, nen initialValues
  // duoc ap lai moi lan va khong dinh vao thu tu effect.
  //
  // PUT /movies/:id la FULL REPLACE va age_rating rong bi doi ve 'P': thieu field
  // nao trong form la xoa cot do khi luu, nen phai liet ke du.
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
      // 400/40001 tra details {json tag: message}: gan thang vao o nhap thay vi
      // toast mot cau chung chung khong chi ro sai o dau.
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

        {/* Backend bind `url`: duong dan tuong doi kieu /media/x.jpg bi tu choi 400. */}
        <Form.Item
          name="poster_url"
          label={t('movie.posterUrl')}
          rules={[{ type: 'url', message: t('movie.urlInvalid') }]}
        >
          <Input placeholder="https://..." maxLength={512} />
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
