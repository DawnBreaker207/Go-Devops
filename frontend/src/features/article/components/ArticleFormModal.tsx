import { App, Button, Drawer, Form, Image, Input, Select, Space, Upload } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { POSTER_ACCEPT, POSTER_MAX_BYTES } from '@/types';
import type { Article, CreateArticlePayload, UpdateArticlePayload } from '@/types';
import { mediaApi } from '@/api/media.api';
import RichTextEditor from '@/components/RichTextEditor';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

const TITLE_MAX = 255;
const SUMMARY_MAX = 500;
const THUMBNAIL_MAX = 1024;

const slugify = (title: string): string =>
  title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

interface FormValues {
  title: string;
  summary?: string;
  thumbnail_url?: string;
  content: string;
  type: 'news' | 'promotion';
}

interface ArticleFormModalProps {
  open: boolean;
  /** null = create, otherwise edit that article. */
  editing: Article | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throws on failure so the drawer stays open and can bind 400/40001 to its inputs. */
  onSubmit: (payload: CreateArticlePayload | UpdateArticlePayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const ArticleFormModal = ({
  open,
  editing,
  confirmLoading,
  onCancel,
  onSubmit,
}: ArticleFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const [uploading, setUploading] = useState(false);

  const watchedThumbnailUrl = Form.useWatch('thumbnail_url', form);
  const watchedTitle = Form.useWatch('title', form);
  const slugPreview = editing ? editing.slug : slugify(watchedTitle ?? '');

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await mediaApi.uploadPoster(file);
      form.setFieldValue('thumbnail_url', result.url);
      await form.validateFields(['thumbnail_url']);
      message.success(t('article.thumbnailUploaded'));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setUploading(false);
    }
  };

  const uploadProps = {
    accept: POSTER_ACCEPT.join(','),
    showUploadList: false as const,
    beforeUpload: (file: File) => {
      if (!POSTER_ACCEPT.includes(file.type as (typeof POSTER_ACCEPT)[number])) {
        message.error(t('article.thumbnailTypeInvalid'));
        return Upload.LIST_IGNORE;
      }
      if (file.size > POSTER_MAX_BYTES) {
        message.error(t('article.thumbnailTooLarge'));
        return Upload.LIST_IGNORE;
      }
      void handleUpload(file);
      return false;
    },
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // Slug: always server-derived from the title (create) or left untouched (edit) - never sent.
      // Status: not editable here - a new article always starts as a draft; publishing/hiding an
      // existing one happens via the quick action in the table, so an edit here leaves it alone.
      const shared = {
        title: values.title.trim(),
        summary: values.summary?.trim() ?? '',
        thumbnail_url: values.thumbnail_url?.trim() ?? '',
        content: values.content,
        type: values.type,
      };
      await onSubmit(editing ? shared : { ...shared, status: 'draft' });
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  return (
    <Drawer
      open={open}
      title={t(editing ? 'article.editTitle' : 'article.createTitle')}
      onClose={onCancel}
      destroyOnHidden
      width={640}
      extra={
        <Space>
          <Button onClick={onCancel}>{t('common.cancel')}</Button>
          <Button type="primary" loading={confirmLoading} onClick={() => void handleOk()}>
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={{
          title: editing?.title ?? '',
          summary: editing?.summary ?? '',
          thumbnail_url: editing?.thumbnail_url ?? '',
          content: editing?.content ?? '',
          type: editing?.type ?? 'news',
        }}
      >
        <Form.Item name="title" label={t('article.title')} rules={[required, { max: TITLE_MAX }]}>
          <Input />
        </Form.Item>

        <Form.Item label={t('article.slug')}>
          <Input value={slugPreview} readOnly disabled />
        </Form.Item>

        <Form.Item name="type" label={t('article.type')} rules={[required]}>
          <Select
            options={[
              { value: 'news', label: t('article.typeNews') },
              { value: 'promotion', label: t('article.typePromotion') },
            ]}
          />
        </Form.Item>

        <Form.Item name="summary" label={t('article.summary')} rules={[{ max: SUMMARY_MAX }]}>
          <Input.TextArea rows={2} />
        </Form.Item>

        <Form.Item
          name="thumbnail_url"
          label={t('article.thumbnail')}
          rules={[{ max: THUMBNAIL_MAX }, { type: 'url', warningOnly: true }]}
        >
          <Input
            placeholder="https://..."
            maxLength={THUMBNAIL_MAX}
            addonAfter={
              <Upload {...uploadProps}>
                <Button
                  type="text"
                  size="small"
                  icon={<UploadOutlined />}
                  loading={uploading}
                  aria-label={t('article.thumbnailUpload')}
                />
              </Upload>
            }
          />
        </Form.Item>

        {watchedThumbnailUrl ? (
          <Form.Item label={t('article.thumbnailPreview')}>
            <Image
              src={watchedThumbnailUrl}
              alt=""
              width={160}
              height={90}
              style={{ objectFit: 'cover', borderRadius: 8 }}
            />
          </Form.Item>
        ) : null}

        <Form.Item name="content" label={t('article.content')} rules={[required]}>
          <RichTextEditor placeholder={t('article.contentPlaceholder')} minHeight={260} />
        </Form.Item>
      </Form>
    </Drawer>
  );
};

export default ArticleFormModal;
