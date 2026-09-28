import { useState } from 'react';
import type { CSSProperties } from 'react';
import {
  Alert,
  App,
  Button,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Steps,
  Typography,
  Upload,
} from 'antd';
import { UploadOutlined, SearchOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
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
import { extractIframeSrc } from '@/features/browse/trailerEmbed';
import { fetchMovieDraft, isTmdbEnabled } from '@/api/tmdb.api';
import { showtimeApi } from '@/api/showtime.api';
import { useTmdbSearch } from '../hooks/useTmdb';
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
  /** Throw on failure so the modal stays open showing the error; resolve lets the page close it. */
  onSubmit: (payload: MoviePayload) => Promise<void>;
}

// antd already paints validateFields errors itself; never toast them again.
const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

const formatMinutes = (min: number): string =>
  min >= 60 ? `${min}p → ${Math.floor(min / 60)}h${min % 60 > 0 ? `${min % 60}m` : ''}` : `${min}p`;

// 3-step wizard keeps each screen under 6 inputs so operators never scroll a wall.
const STEP_TMDB = 0;
const STEP_INFO = 1;
const STEP_MEDIA = 2;

const INFO_FIELDS: Array<keyof FormValues> = [
  'title',
  'genre',
  'director',
  'cast',
  'duration',
  'release_date',
];

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

  const hasTmdbStep = isTmdbEnabled() && movie === null;
  const [step, setStep] = useState(hasTmdbStep ? STEP_TMDB : STEP_INFO);

  // Steps stay mounted (display:none), so reset the step during render, not in an effect.
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setStep(hasTmdbStep ? STEP_TMDB : STEP_INFO);
  }

  type ImageField = 'poster_url' | 'backdrop_url';
  const [uploadingField, setUploadingField] = useState<ImageField | null>(null);
  const [tmdbQuery, setTmdbQuery] = useState('');
  const [filling, setFilling] = useState(false);
  const [reuploadingField, setReuploadingField] = useState<ImageField | null>(null);

  const tmdbSearch = useTmdbSearch(hasTmdbStep ? tmdbQuery : '');
  const posterUrl = Form.useWatch('poster_url', form);
  const backdropUrl = Form.useWatch('backdrop_url', form);
  const duration = Form.useWatch('duration', form);

  // Pre-lock duration/status ahead of the 409 while future showtimes exist.
  const futureShows = useQuery({
    queryKey: ['movie-future-showtimes', movie?.id],
    queryFn: () =>
      showtimeApi.list({
        movie_id: movie?.id,
        from: dayjs().format('YYYY-MM-DD'),
        page_size: 1,
      }),
    enabled: open && movie !== null,
    staleTime: 60_000,
    retry: 1,
  });
  const hasFutureShows = (futureShows.data?.meta.total ?? 0) > 0;
  const lockDuration = movie !== null && hasFutureShows;
  const lockStatus = movie?.status === 'showing' && hasFutureShows;

  const handleUpload = async (file: File, field: ImageField) => {
    setUploadingField(field);
    try {
      const result = await mediaApi.uploadPoster(file);
      form.setFieldValue(field, result.url);
      // Programmatic sets skip typing, so re-run the url rule by hand.
      await form.validateFields([field]);
      message.success(t('movie.posterUploaded'));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setUploadingField(null);
    }
  };

  // Pull TMDB hotlinks into /media to drop the outside CDN dependency; fall back to hotlink on failure.
  const handleReupload = async (url: string, field: ImageField) => {
    setReuploadingField(field);
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const blob = await res.blob();
      const ext = blob.type.includes('png') ? 'png' : blob.type.includes('webp') ? 'webp' : 'jpg';
      await handleUpload(
        new File([blob], `tmdb-${field}.${ext}`, { type: blob.type || 'image/jpeg' }),
        field
      );
    } catch {
      form.setFieldValue(field, url);
      await form.validateFields([field]);
      message.warning(t('movie.posterHotlinkFallback'));
    } finally {
      setReuploadingField(null);
    }
  };

  const fillFromTmdb = async (tmdbId: number) => {
    setFilling(true);
    try {
      const draft = await fetchMovieDraft(tmdbId);
      form.setFieldsValue({
        ...(draft.title ? { title: draft.title } : null),
        ...(draft.genre ? { genre: draft.genre } : null),
        ...(draft.duration ? { duration: draft.duration } : null),
        ...(draft.director ? { director: draft.director } : null),
        ...(draft.cast ? { cast: draft.cast } : null),
        ...(draft.release_date ? { release_date: dayjs(draft.release_date, 'YYYY-MM-DD') } : null),
        ...(draft.description ? { description: draft.description } : null),
        ...(draft.poster_url ? { poster_url: draft.poster_url } : null),
        ...(draft.backdrop_url ? { backdrop_url: draft.backdrop_url } : null),
        ...(draft.trailer_url ? { trailer_url: draft.trailer_url } : null),
      });
      setTmdbQuery('');
      setStep(STEP_INFO);
      message.success(t('movie.tmdbFilled'));
      if (!draft.trailer_url) message.info(t('movie.tmdbNoTrailer'));
    } catch (error) {
      message.error(errorMessage(error, t('movie.tmdbFailed')));
    } finally {
      setFilling(false);
    }
  };

  // PUT /movies/:id is a full replace, so list every field including backdrop_url.
  const initialValues: Partial<FormValues> = movie
    ? {
        title: movie.title,
        genre: movie.genre,
        duration: movie.duration,
        director: movie.director,
        description: movie.description,
        poster_url: movie.poster_url,
        backdrop_url: movie.backdrop_url,
        trailer_url: movie.trailer_url,
        cast: movie.cast,
        age_rating: movie.age_rating,
        status: movie.status,
        release_date: dayjs(movie.release_date),
      }
    : { status: 'draft' as MovieStatus, duration: 90, age_rating: 'P' };

  const goNext = async () => {
    try {
      await form.validateFields(INFO_FIELDS);
      setStep(STEP_MEDIA);
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit({
        ...values,
        release_date: values.release_date.format('YYYY-MM-DD'),
        // Iframe paste is just an entry aid; the backend always stores the clean URL from src.
        trailer_url: values.trailer_url
          ? extractIframeSrc(values.trailer_url.trim())
          : values.trailer_url,
      });
    } catch (error) {
      if (isFormValidationError(error)) {
        // A media-step error while standing on info jumps there to be seen.
        setStep(STEP_MEDIA);
        return;
      }
      // 400/40001 returns details {json tag: message}: attach straight onto the input instead of a generic toast.
      if (applyApiFieldErrors(form, error)) {
        setStep(STEP_MEDIA);
        return;
      }
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  const stepItems = [
    ...(hasTmdbStep ? [{ title: t('movie.stepTmdb') }] : []),
    { title: t('movie.stepInfo') },
    { title: t('movie.stepMedia') },
  ];
  const stepIndex = hasTmdbStep ? step : step - STEP_INFO;

  // Stock antd uploaders send no Authorization header, so beforeUpload + handleUpload instead.
  const imageUploadProps = (field: 'poster_url' | 'backdrop_url') => ({
    accept: POSTER_ACCEPT.join(','),
    showUploadList: false as const,
    beforeUpload: (file: File) => {
      if (!POSTER_ACCEPT.includes(file.type as (typeof POSTER_ACCEPT)[number])) {
        message.error(t('movie.posterTypeInvalid'));
        return Upload.LIST_IGNORE;
      }
      if (file.size > POSTER_MAX_BYTES) {
        message.error(t('movie.posterTooLarge'));
        return Upload.LIST_IGNORE;
      }
      void handleUpload(file, field);
      return false;
    },
  });

  const imagePreview = (
    url: string | undefined,
    field: 'poster_url' | 'backdrop_url',
    imgStyle: CSSProperties
  ) =>
    url ? (
      <div
        style={{
          marginTop: -12,
          marginBottom: 16,
          display: 'flex',
          gap: 12,
          alignItems: 'center',
        }}
      >
        <img
          src={url}
          alt=""
          style={{ borderRadius: 6, objectFit: 'cover', ...imgStyle }}
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = 'none';
          }}
        />
        <Button
          size="small"
          loading={reuploadingField === field}
          onClick={() => void handleReupload(url, field)}
        >
          {t('movie.posterReupload')}
        </Button>
      </div>
    ) : null;

  return (
    <Modal
      open={open}
      title={movie ? t('movie.editTitle') : t('movie.createTitle')}
      onCancel={onCancel}
      destroyOnHidden
      width={760}
      footer={
        step === STEP_MEDIA ? (
          <>
            <Button onClick={() => setStep(STEP_INFO)}>{t('common.back')}</Button>
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
            <Button type="primary" loading={confirmLoading} onClick={() => void handleSave()}>
              {t('common.save')}
            </Button>
          </>
        ) : step === STEP_TMDB ? (
          <>
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={() => setStep(STEP_INFO)}>
              {t('common.skip')}
            </Button>
          </>
        ) : (
          <>
            {hasTmdbStep ? (
              <Button onClick={() => setStep(STEP_TMDB)}>{t('common.back')}</Button>
            ) : null}
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={() => void goNext()}>
              {t('common.next')}
            </Button>
          </>
        )
      }
    >
      <Steps current={stepIndex} items={stepItems} style={{ marginBottom: 20 }} />

      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        {/* Step giu mounted + an display none vi unmount mat gia tri form store. */}
        <div style={{ display: step === STEP_TMDB ? '' : 'none' }}>
          <Input.Search
            placeholder={t('movie.tmdbSearchPlaceholder')}
            allowClear
            enterButton={<SearchOutlined />}
            loading={tmdbSearch.isFetching || filling}
            value={tmdbQuery}
            onChange={(e) => setTmdbQuery(e.target.value)}
            onSearch={(v) => setTmdbQuery(v)}
          />
          {tmdbSearch.error ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 8 }}
              message={errorMessage(tmdbSearch.error, t('movie.tmdbFailed'))}
            />
          ) : null}
          {(tmdbSearch.data ?? []).length > 0 ? (
            <div style={{ marginTop: 8, maxHeight: 320, overflowY: 'auto' }}>
              {tmdbSearch.data?.map((r) => (
                <div
                  key={r.id}
                  style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '6px 0' }}
                >
                  {r.poster_thumb ? (
                    <img src={r.poster_thumb} alt="" width={40} style={{ borderRadius: 4 }} />
                  ) : null}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 600 }}>
                      {r.title}
                      {r.release_date ? ` (${r.release_date.slice(0, 4)})` : ''}
                    </div>
                    {r.overview ? (
                      <Typography.Text type="secondary" ellipsis style={{ fontSize: 12 }}>
                        {r.overview}
                      </Typography.Text>
                    ) : null}
                  </div>
                  <Button size="small" loading={filling} onClick={() => void fillFromTmdb(r.id)}>
                    {t('movie.tmdbUse')}
                  </Button>
                </div>
              ))}
            </div>
          ) : null}
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('movie.tmdbAttribution')}{' '}
            <a href="https://www.themoviedb.org/" target="_blank" rel="noreferrer">
              TMDB
            </a>
            {' · '}
            {t('movie.tmdbSkipHint')}
          </Typography.Text>
        </div>

        <div style={{ display: step === STEP_INFO ? '' : 'none' }}>
          <>
            <Form.Item name="title" label={t('movie.name')} rules={[required]}>
              <Input placeholder="Inception" maxLength={255} />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="genre" label={t('movie.genre')} rules={[required]}>
                  <Input placeholder="Sci-Fi" maxLength={100} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="director" label={t('movie.director')} rules={[required]}>
                  <Input placeholder="Christopher Nolan" maxLength={255} />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item name="cast" label={t('movie.cast')}>
              <Input.TextArea rows={2} maxLength={2000} placeholder="Leonardo DiCaprio, ..." />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="duration"
                  label={t('movie.duration')}
                  rules={[required]}
                  extra={
                    typeof duration === 'number' && duration > 0
                      ? formatMinutes(duration)
                      : lockDuration
                        ? t('movie.durationLocked')
                        : undefined
                  }
                >
                  <InputNumber
                    min={1}
                    max={600}
                    disabled={lockDuration}
                    addonAfter={t('common.minutes')}
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="release_date" label={t('movie.releaseDate')} rules={[required]}>
                  <DatePicker style={{ width: '100%' }} format="DD/MM/YYYY" />
                </Form.Item>
              </Col>
            </Row>
          </>
        </div>

        <div style={{ display: step === STEP_MEDIA ? '' : 'none' }}>
          <>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="age_rating" label={t('movie.ageRating')}>
                  <Select options={MOVIE_AGE_RATINGS.map((value) => ({ value, label: value }))} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="status"
                  label={t('movie.status')}
                  rules={[required]}
                  extra={lockStatus ? t('movie.statusLocked') : t('movie.statusHint')}
                >
                  <Select
                    disabled={lockStatus}
                    options={[
                      { value: 'draft', label: t('movie.statusDraft') },
                      { value: 'coming_soon', label: t('movie.statusComing_soon') },
                      { value: 'showing', label: t('movie.statusShowing') },
                      { value: 'ended', label: t('movie.statusEnded') },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            {/* Backend 400 path tuong doi /media/x.jpg nen chi nhan url tuyet doi. */}
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
                  <Upload {...imageUploadProps('poster_url')}>
                    <Button
                      type="text"
                      size="small"
                      icon={<UploadOutlined />}
                      loading={uploadingField === 'poster_url'}
                      aria-label={t('movie.posterUpload')}
                    />
                  </Upload>
                }
              />
            </Form.Item>
            {imagePreview(posterUrl, 'poster_url', { width: 72 })}

            <Form.Item
              name="backdrop_url"
              label={t('movie.backdropUrl')}
              extra={t('movie.backdropUploadHint')}
              rules={[{ type: 'url', message: t('movie.urlInvalid') }]}
            >
              <Input
                placeholder="https://..."
                maxLength={512}
                addonAfter={
                  <Upload {...imageUploadProps('backdrop_url')}>
                    <Button
                      type="text"
                      size="small"
                      icon={<UploadOutlined />}
                      loading={uploadingField === 'backdrop_url'}
                      aria-label={t('movie.posterUpload')}
                    />
                  </Upload>
                }
              />
            </Form.Item>
            {imagePreview(backdropUrl, 'backdrop_url', { width: 128, aspectRatio: '16 / 9' })}

            <Form.Item
              name="trailer_url"
              label={t('movie.trailerUrl')}
              extra={t('movie.trailerUrlHint')}
              rules={[
                {
                  validator: (_, value?: string) => {
                    if (!value) return Promise.resolve();
                    try {
                      new URL(extractIframeSrc(value.trim()));
                      return Promise.resolve();
                    } catch {
                      return Promise.reject(new Error(t('movie.urlInvalid')));
                    }
                  },
                },
              ]}
            >
              <Input.TextArea
                placeholder="https://www.youtube.com/watch?v=... hoặc <iframe ...>"
                autoSize={{ minRows: 1, maxRows: 4 }}
                maxLength={2000}
              />
            </Form.Item>

            <Form.Item name="description" label={t('movie.description')}>
              <Input.TextArea rows={3} maxLength={5000} showCount />
            </Form.Item>
          </>
        </div>
      </Form>
    </Modal>
  );
};

export default MovieFormModal;
