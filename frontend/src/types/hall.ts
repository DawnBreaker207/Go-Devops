/**
 * Mirror cua internal/dto/hall.go. Quy tac optionality: field Go kieu gia tri
 * khong co `omitempty` la BAT BUOC; co `omitempty` hoac con tro la tuy chon.
 *
 * Ba cho de nham nhat cua nhom endpoint nay:
 * 1. Gia doc ve la MANG, ghi len la MAP - va hai chieu con sap xep khac nhau.
 * 2. Danh sach ghe la MANG TRAN, khong phan trang, khong co meta.
 * 3. So o luoi KHONG bang rows * seats_per_row khi co ghe col_span=2.
 */

/** models.AllSeatTypes. Khong endpoint nao tra ve danh sach nay - FE phai tu giu. */
export type SeatType = 'standard' | 'vip' | 'couple' | 'recliner';

/**
 * Dung thu tu cua models.AllSeatTypes, cung la thu tu PUT /prices tra ve.
 * GET /prices lai sap theo alphabet (couple, recliner, standard, vip), nen moi
 * cho hien thi deu phai sap lai theo hang so nay thay vi tin thu tu cua server.
 */
export const SEAT_TYPES: readonly SeatType[] = ['standard', 'vip', 'couple', 'recliner'] as const;

export type ScreenPosition = 'front' | 'back';
export const SCREEN_POSITIONS: readonly ScreenPosition[] = ['front', 'back'] as const;

export type HallTemplateName = 'small' | 'medium' | 'large';

/** SQL: col_span SMALLINT CHECK (col_span IN (1,2)). */
export type ColSpan = 1 | 2;

/** GET /api/v1/admin/halls -> PagedData<Hall>. Khong field nao omitempty. */
export interface Hall {
  id: string;
  name: string;
  rows: number;
  seats_per_row: number;
  screen_position: ScreenPosition;
  /** JSONB NOT NULL DEFAULT '[]' + normalizeHallJSON nen khong bao gio null.
   *  Chi la goi y ve: backend khong dung no de tinh gi ca, va CHI kiem tra do
   *  dai <= 49 chu khong kiem tra gia tri, nen co the co so 0/am/vuot cot. */
  aisle_after_cols: number[];
  /** Phong active=false bi tu choi khi tao suat chieu moi (409). */
  active: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * dto.SeatResponse - GET /admin/halls/:id/seats (MANG TRAN, khong phan trang).
 * `label` do backend ghep tu row_label + col_number, khong luu trong DB va
 * khong ghi len duoc. Khoa luoi that su la (row_label, col_number).
 */
export interface Seat {
  id: string;
  hall_id: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  /** O luoi co ton tai nhung khong ban duoc: loi di, cho xe lan, cot nha.
   *  Van duoc tinh gia va van co ban ghi showtime_seats, chi la giu cho la
   *  khong ban duoc - backend tra ErrSeatNotSellable khi co dat. */
  is_gap: boolean;
  /** Ghe col_span=2 chiem cot N va N+1, va KHONG co ban ghi ghe nao o N+1.
   *  So cot vi vay co lo hong; tuyet doi khong ve luoi theo chi so mang. */
  col_span: ColSpan;
}

/** dto.HallPriceResponse - phia DOC (mang) va phia echo cua PUT (cung mang). */
export interface HallPrice {
  seat_type: SeatType;
  /** int64 VND nguyen. 0 nghia la CHUA CAU HINH, khong phai mien phi. */
  price: number;
}

/** dto.HallTemplateResponse - GET /admin/hall-templates (mang tran, 3 muc). */
export interface HallTemplate {
  name: HallTemplateName;
  rows: number;
  seats_per_row: number;
  /** Da tru ghe is_gap va da tinh ca ghe col_span=2 (nen < rows * seats_per_row). */
  seat_count: number;
  /** MOT PHAN: loai ghe nao dem duoc 0 thi vang mat khoi map. */
  seat_count_by_type: Partial<Record<SeatType, number>>;
}

/**
 * dto.HallRequest - dung chung cho POST /admin/halls VA PUT /admin/halls/:id/layout.
 * Tren /layout, `name` va `prices` van BAT BUOC theo binding nhung service
 * khong doc ca hai.
 */
export interface HallPayload {
  name: string;
  template?: HallTemplateName;
  rows?: number;
  seats_per_row?: number;
  /** {loai ghe: SO THU TU hang dang chuoi}, vi du {"vip": ["6","7"]} = hang F, G.
   *  La so hang chu khong phai nhan hang. Hang khong liet ke thi la standard. */
  seat_types?: Partial<Record<SeatType, string[]>>;
  /** Nhan ghe bi bien thanh o trong, vi du ["D5","D6"]. Toi da 200. */
  gaps?: string[];
  /** Nhan ghe la diem neo cua ghe doi 2 cot, vi du ["D3"] = mot ghe phu D3-D4.
   *  Toi da 100. */
  spans?: string[];
  screen_position?: ScreenPosition;
  aisle_after_cols?: number[];
  /** BAT BUOC. POST doi du 4 loai ghe voi gia > 0; PUT /layout bo qua hoan toan. */
  prices: Record<SeatType, number>;
}

/**
 * dto.UpdateHallRequest - PUT /admin/halls/:id. Bo trong field nao thi giu
 * nguyen field do. KHONG BAO GIO dung toi ghe.
 */
export interface UpdateHallPayload {
  name?: string;
  screen_position?: ScreenPosition;
  /** KHONG phai con tro ben Go: bo trong de giu, gui [] de XOA HET loi di. */
  aisle_after_cols?: number[];
  active?: boolean;
}

/** dto.CloneHallRequest - POST /admin/halls/:id/clone. */
export interface CloneHallPayload {
  name: string;
  /** Khong co binding tag, mac dinh false. false = phong moi KHONG co gia nao. */
  copy_prices?: boolean;
}

/** dto.PriceRequest - PUT /admin/halls/:id/prices. Phia GHI la MAP, du 4 loai. */
export interface PricePayload {
  prices: Record<SeatType, number>;
}

/**
 * dto.SeatSelector - dung MOT va chi mot field duoc khac rong. Binding cua Go
 * khong bat duoc dieu do (SeatSelector la struct gia tri), service moi bat, nen
 * loi ve la 400/40001 details {selector: "exactly one of ..."}.
 */
export interface SeatSelector {
  /** Toi da 500. Duoc viet hoa khi so khop nhung KHONG duoc trim: mot nhan thua
   *  khoang trang se khong khop ghe nao va lam hong ca lo. Trim o FE. */
  labels?: string[];
  /** Toi da 50. Duoc trim VA viet hoa. */
  rows?: string[];
  /** Toi da 50, so cot chinh xac. */
  cols?: number[];
  /** Hinh chu nhat dong ca hai dau, vi du "A1:C4". */
  range?: string;
}

/** dto.SeatChange. Thieu ca seat_type lan is_gap van hop le - mot thay doi rong. */
export interface SeatChange {
  selector: SeatSelector;
  seat_type?: SeatType;
  is_gap?: boolean;
}

/** dto.BulkSeatUpdateRequest - PATCH /admin/halls/:id/seats. 1..50 thay doi. */
export interface BulkSeatUpdatePayload {
  changes: SeatChange[];
}

/** dto.SeatUpdateRequest - PUT /admin/halls/:id/seats/:seatId. */
export interface SeatUpdatePayload {
  seat_type?: SeatType;
  is_gap?: boolean;
}
