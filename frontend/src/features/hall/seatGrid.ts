import type { Hall, Seat } from '@/types';

/**
 * Hinh dang TOI THIEU cua mot ghe de ve duoc luoi.
 *
 * Co hai kieu ghe khac nhau di qua day: `Seat` (luoi vat ly cua phong, man
 * admin) va `SeatMapSeat` (luoi cua mot suat chieu, man khach - them status,
 * price, showtime_seat_id va BO hall_id). Ca hai deu thoa hinh dang nay, nen
 * toan bo phan toan hoc dung chung duoc thay vi viet lai lan thu hai.
 */
export interface GridSeat {
  row_label: string;
  col_number: number;
  col_span: number;
  is_gap: boolean;
}

/**
 * Toan hoc cua luoi ghe, tach rieng khoi React vi day la cho de sai nhat cua ca
 * man hinh va la cho duy nhat dang viet test.
 *
 * Hai su that tu backend chi phoi toan bo file nay:
 *
 * 1. Ghe `col_span = 2` chiem cot N VA N+1, va KHONG co ban ghi ghe nao o cot
 *    N+1 (generateSeats bo qua cot da bi nuot). Vi vay so cot co LO HONG, va
 *    khong bao gio duoc ve luoi theo chi so mang.
 * 2. `aisle_after_cols` chi la goi y ve. Backend chi kiem tra do dai mang
 *    (<= 49) chu KHONG kiem tra gia tri, nen no co the chua 0, so am, hoac so
 *    lon hon seats_per_row. Phia ve phai tu loc.
 */

/** Chieu rong mot o ghe, tinh bang px. */
export const SEAT_SIZE = 30;
/** Be rong cua khe loi di chen giua hai cot. */
export const AISLE_WIDTH = 18;

export interface SeatRow<T extends GridSeat = Seat> {
  rowLabel: string;
  seats: T[];
}

/**
 * Gom ghe theo hang, GIU NGUYEN thu tu backend tra ve (ORDER BY row_index,
 * col_number). Khong tu sap xep lai theo row_label: sap chuoi se sai tu hang AA
 * tro di (AA dung truoc B theo alphabet nhung la hang thu 27).
 */
export const groupSeatsByRow = <T extends GridSeat>(seats: T[]): SeatRow<T>[] => {
  const rows: SeatRow<T>[] = [];
  const index = new Map<string, SeatRow<T>>();
  seats.forEach((seat) => {
    let row = index.get(seat.row_label);
    if (!row) {
      row = { rowLabel: seat.row_label, seats: [] };
      index.set(seat.row_label, row);
      rows.push(row);
    }
    row.seats.push(seat);
  });
  return rows;
};

/**
 * Chi giu nhung vi tri loi di thuc su ve duoc: so nguyen trong khoang
 * 1..seatsPerRow-1, khong trung nhau. Loi di sau cot cuoi cung khong ve gi ca
 * nen bi loai luon.
 */
export const normalizeAisles = (aisleAfterCols: number[], seatsPerRow: number): number[] => {
  const seen = new Set<number>();
  aisleAfterCols.forEach((col) => {
    if (Number.isInteger(col) && col >= 1 && col < seatsPerRow) seen.add(col);
  });
  return [...seen].sort((a, b) => a - b);
};

export interface GridLayout {
  /** Gia tri cho CSS `grid-template-columns`. */
  templateColumns: string;
  /** Vach luoi bat dau cua mot so cot (CSS grid dem tu 1). */
  lineOf: (col: number) => number;
  /** So TRACK mot ghe chiem - khong bang col_span khi co khe loi di ken giua. */
  spanOf: (col: number, colSpan: number) => number;
  aisles: number[];
}

/**
 * Loi di duoc ve bang mot TRACK RIENG chu khong phai margin cua o ghe: neu lay
 * margin thi o ghe bi bop lai trong khi be rong cot van giu nguyen, va luoi
 * lech dan ve ben phai.
 */
export const buildGridLayout = (seatsPerRow: number, aisleAfterCols: number[]): GridLayout => {
  const aisles = normalizeAisles(aisleAfterCols, seatsPerRow);
  const aisleSet = new Set(aisles);

  const tracks: string[] = [];
  for (let col = 1; col <= seatsPerRow; col += 1) {
    tracks.push(`${SEAT_SIZE}px`);
    if (aisleSet.has(col)) tracks.push(`${AISLE_WIDTH}px`);
  }

  const lineOf = (col: number): number => {
    // Vach = 1 + so track dung truoc no = 1 + (col-1) o ghe + so loi di da chen.
    let aislesBefore = 0;
    aisles.forEach((aisle) => {
      if (aisle < col) aislesBefore += 1;
    });
    return col + aislesBefore;
  };

  const spanOf = (col: number, colSpan: number): number => {
    // Ghe doi phu cot col va col+1. Neu loi di roi dung giua hai cot do thi ghe
    // phai keo qua ca track loi di, tuc 3 track chu khong phai 2.
    if (colSpan <= 1) return 1;
    return colSpan + (aisleSet.has(col) ? 1 : 0);
  };

  return { templateColumns: tracks.join(' '), lineOf, spanOf, aisles };
};

/**
 * Suc chua BAN DUOC. Khong co field nao cua backend noi con so nay:
 * `rows * seats_per_row` la so O LUOI chu khong phai so ghe (ghe doi nuot mot
 * cot, ghe is_gap khong ban duoc), va moi truy van bao cao ben backend deu dem
 * bang `COUNT(*) FILTER (WHERE NOT is_gap)`. Tinh giong het o day.
 */
export const sellableCapacity = (seats: GridSeat[]): number =>
  seats.reduce((total, seat) => (seat.is_gap ? total : total + 1), 0);

/**
 * `labels` cua selector duoc backend viet hoa nhung KHONG trim (chi `rows` moi
 * duoc trim). Mot nhan thua khoang trang se khong khop ghe nao, va "khong khop
 * ghe nao" la loi 400 lam ROLLBACK CA LO - ke ca nhung thay doi hop le dung
 * canh no. Vi vay moi nhan deu phai duoc lam sach o day truoc khi gui.
 */
export const cleanSeatLabel = (label: string): string => label.trim().toUpperCase();

/** Toi da cua backend: 50 thay doi moi lan goi, 500 nhan moi thay doi. */
export const MAX_CHANGES_PER_CALL = 50;
export const MAX_LABELS_PER_CHANGE = 500;

/**
 * Cat danh sach nhan thanh nhieu `SeatChange` de khong vuot tran 500 nhan.
 * Tra ve nhieu lo neu so thay doi vuot qua 50 - moi lo la mot lan goi API rieng,
 * vi backend chay ca lo trong MOT transaction va lo qua lon thi ca lo cung hong.
 */
export const chunkLabels = (labels: string[]): string[][][] => {
  const changes: string[][] = [];
  for (let i = 0; i < labels.length; i += MAX_LABELS_PER_CHANGE) {
    changes.push(labels.slice(i, i + MAX_LABELS_PER_CHANGE));
  }
  const batches: string[][][] = [];
  for (let i = 0; i < changes.length; i += MAX_CHANGES_PER_CALL) {
    batches.push(changes.slice(i, i + MAX_CHANGES_PER_CALL));
  }
  return batches;
};

/**
 * So cot THUC SU phai ve. Khong tin thang `hall.seats_per_row`: no co the nho
 * hon luoi ghe that.
 *
 * KHONG co rang buoc nao trong CSDL buoc `halls.rows`/`seats_per_row` khop voi
 * bang `seats` - hai bang duoc ghi bang hai lenh khac nhau trong cung mot
 * transaction. Da tung lech that: `RegenerateLayout` ghi qua `UpdateHall`, ma
 * whitelist cot cua no khong co rows/seats_per_row, nen doi phong 4x5 thanh 6x8
 * sinh 48 ghe toi cot 8 con bang halls van noi 4x5 mai mai (da va o BE ngay
 * 2026-09-18 bang `UpdateHallLayout` + test hoi quy T76b).
 *
 * Van ve theo du lieu ghe that chu khong theo con so phong khai bao: luoi khi do
 * khong bao gio bi cat, va `declaredMismatch` bao cho nguoi truc biet hai ben
 * lech thay vi ve am tham mot so do sai.
 */
export const widestColumn = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  seats.reduce((max, s) => Math.max(max, s.col_number + s.col_span - 1), 0);

/**
 * Bien the cho man KHACH. `GET /shows/:id/seats` khong tra ve rows /
 * seats_per_row cua phong - chi co ghe - nen so cot phai suy hoan toan tu du
 * lieu ghe.
 */
export const seatsPerRowFromSeats = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  widestColumn(seats);

export const renderedSeatsPerRow = (hall: Hall, seats: Seat[]): number =>
  Math.max(widestColumn(seats), hall.seats_per_row);

/** Mo ta luoi cho phan tom tat cua man hinh. */
export interface GridSummary {
  gridCells: number;
  seatCount: number;
  sellable: number;
  gaps: number;
  doubleSeats: number;
  /** true khi `rows`/`seats_per_row` cua phong khong khop luoi ghe that. */
  declaredMismatch: boolean;
  actualRows: number;
  actualSeatsPerRow: number;
}

export const summarizeGrid = (hall: Hall, seats: Seat[]): GridSummary => {
  const actualRows = groupSeatsByRow(seats).length;
  const actualSeatsPerRow = widestColumn(seats);
  return {
    gridCells: hall.rows * hall.seats_per_row,
    seatCount: seats.length,
    sellable: sellableCapacity(seats),
    gaps: seats.filter((s) => s.is_gap).length,
    doubleSeats: seats.filter((s) => s.col_span === 2).length,
    // Phong chua co ghe nao thi khong coi la lech - do la trang thai khac.
    declaredMismatch:
      seats.length > 0 && (actualRows !== hall.rows || actualSeatsPerRow !== hall.seats_per_row),
    actualRows,
    actualSeatsPerRow,
  };
};
