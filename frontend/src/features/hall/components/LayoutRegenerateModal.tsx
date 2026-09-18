import { App, Alert, Form, Input, InputNumber, Modal, Select, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  Hall,
  HallPayload,
  HallPrice,
  HallTemplateName,
  ScreenPosition,
  SeatType,
} from '@/types';
import { SCREEN_POSITIONS, SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { useHallTemplates, useRegenerateLayout } from '../hooks/useHalls';
import { MAX_ROWS, MAX_SEATS_PER_ROW } from '../constants';

interface FormValues {
  template?: HallTemplateName | '';
  rows?: number;
  seats_per_row?: number;
  screen_position: ScreenPosition;
  aisle_after_cols: string;
}

interface LayoutRegenerateModalProps {
  open: boolean;
  hall: Hall;
  /** Gia hien tai, dung de dap lai vao body - xem ghi chu trong handleOk. */
  prices: HallPrice[];
  onCancel: () => void;
  onDone: () => void;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/** "4, 10" -> [4, 10]. Bo moi thu khong phai so duong. */
const parseAisles = (raw: string): number[] =>
  raw
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((value) => Number.isInteger(value) && value > 0);

export const LayoutRegenerateModal = ({
  open,
  hall,
  prices,
  onCancel,
  onDone,
}: LayoutRegenerateModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const templates = useHallTemplates();
  const regenerate = useRegenerateLayout();

  const template = Form.useWatch('template', form);
  const usingTemplate = Boolean(template);

  // Nap bang initialValues chu khong setFieldsValue trong effect - StrictMode +
  // preserve={false} se xoa mat gia tri. Xem .claude/rules/pages-components.md.
  //
  // screen_position va aisle_after_cols BAT BUOC phai duoc nap san: service gan
  // thang `current.ScreenPosition = req.ScreenPosition`, nen bo trong la reset ve
  // 'front' va xoa sach loi di - mat du lieu am tham.
  const initialValues: FormValues = {
    template: '',
    rows: hall.rows,
    seats_per_row: hall.seats_per_row,
    screen_position: hall.screen_position,
    aisle_after_cols: hall.aisle_after_cols.join(', '),
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();

      /**
       * `name` va `prices` van BAT BUOC theo binding cua dto.HallRequest du
       * RegenerateLayout khong doc field nao trong hai - endpoint nay dung chung
       * DTO voi POST /admin/halls. Gui lai ten hien tai va bo gia hien tai; loai
       * nao chua co gia thi dien 1 cho du 4 khoa, gia tri khong di den dau ca.
       */
      const priceMap = SEAT_TYPES.reduce<Record<SeatType, number>>(
        (acc, type) => {
          acc[type] = prices.find((p) => p.seat_type === type)?.price ?? 1;
          return acc;
        },
        {} as Record<SeatType, number>
      );

      /**
       * Chon mau thi PHAI BO TRONG screen_position va aisle_after_cols.
       *
       * `resolveLayout` ben backend chi dien tu mau vao nhung field con trong:
       * `if len(req.AisleAfterCols) == 0 { req.AisleAfterCols = t.aisleAfterCols }`.
       * Form nay nap san hai field do tu phong HIEN TAI, nen neu cu gui di thi
       * chung luon khac rong va luon thang mau - loi di cua mau bi vut im lang,
       * dung cai ma canh bao ngay trong modal noi la se lay theo mau.
       */
      const payload: HallPayload = values.template
        ? { name: hall.name, prices: priceMap, template: values.template }
        : {
            name: hall.name,
            prices: priceMap,
            rows: values.rows,
            seats_per_row: values.seats_per_row,
            // Khong co mau thi hai field nay BAT BUOC phai gui: RegenerateLayout
            // gan thang `current.ScreenPosition = req.ScreenPosition`, bo trong
            // la reset ve 'front' va xoa sach loi di.
            screen_position: values.screen_position,
            aisle_after_cols: parseAisles(values.aisle_after_cols),
          };

      await regenerate.mutateAsync({ id: hall.id, payload });
      message.success(t('hall.regenerateSuccess'));
      onDone();
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <Modal
      open={open}
      title={t('hall.regenerateTitle')}
      okText={t('hall.regenerateConfirm')}
      okButtonProps={{ danger: true }}
      cancelText={t('common.cancel')}
      confirmLoading={regenerate.isPending}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={560}
    >
      {/*
        Canh bao phai hien TRUOC khi nguoi dung dien, chu khong doi 409 tra ve:
        409 chi den sau khi ho da go xong ca cai form.
      */}
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        message={t('hall.regenerateWarnTitle')}
        description={t('hall.regenerateWarnBody')}
      />

      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item name="template" label={t('hall.template')} extra={t('hall.templateHint')}>
          <Select
            loading={templates.isFetching}
            options={[
              { value: '', label: t('hall.templateNone') },
              ...(templates.data ?? []).map((tpl) => ({
                value: tpl.name,
                label: `${t(`hall.template_${tpl.name}`)} — ${tpl.rows}×${tpl.seats_per_row}, ${t(
                  'hall.templateSeats',
                  { count: tpl.seat_count }
                )}`,
              })),
            ]}
          />
        </Form.Item>

        {/*
          Chon mau thi backend tu dien rows/seats_per_row VA ca seat_types, gaps,
          spans, aisle_after_cols cua mau do - khong phai cua phong hien tai.
        */}
        {usingTemplate ? (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('hall.templateOverrides')}
          />
        ) : (
          <>
            <Form.Item
              name="rows"
              label={t('hall.rows')}
              rules={[{ required: true, message: t('common.requiredField') }]}
            >
              <InputNumber min={1} max={MAX_ROWS} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item
              name="seats_per_row"
              label={t('hall.seatsPerRow')}
              rules={[{ required: true, message: t('common.requiredField') }]}
            >
              <InputNumber min={1} max={MAX_SEATS_PER_ROW} style={{ width: '100%' }} />
            </Form.Item>
          </>
        )}

        {/* An khi da chon mau: hai field nay khong duoc gui nua, nen de chung
            hien ra voi gia tri cu la noi doi voi nguoi dung. */}
        {usingTemplate ? null : (
          <>
            <Form.Item name="screen_position" label={t('hall.screenPosition')}>
              <Select
                options={SCREEN_POSITIONS.map((value) => ({
                  value,
                  label: t(`hall.screen_${value}`),
                }))}
              />
            </Form.Item>

            <Form.Item
              name="aisle_after_cols"
              label={t('hall.aisles')}
              extra={t('hall.aislesHint')}
            >
              <Input placeholder="4, 10" />
            </Form.Item>
          </>
        )}
      </Form>

      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('hall.regenerateScopeNote')}
      </Typography.Text>
    </Modal>
  );
};

export default LayoutRegenerateModal;
