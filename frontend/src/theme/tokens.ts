/**
 * Design token cua CinemaProject.
 *
 * NGUON: file Figma "Movie Ticket Booking Website (Community)" - xem
 * `.claude/context/figma.md` de biet node-id tung frame va cach mo lai.
 * Moi gia tri danh dau MEASURED duoc doc truc tiep tu pixel cua anh chup frame
 * (mau troi cua mot vung), khong phai uoc luong bang mat. Gia tri danh dau
 * DERIVED la do file nay tu suy ra vi thiet ke khong dinh nghia - moi cho deu
 * noi ro suy ra tu dau.
 *
 * File nay KHONG import antd va khong phu thuoc React: no la nguon su that duy
 * nhat. `src/theme/index.ts` do no sang token cua antd, `src/index.css` mirror
 * phan ma CSS can, va `src/__tests__/design-tokens.test.ts` fail neu hai ben lech.
 */

/* -------------------------------------------------------------------------- */
/* 1. Brand                                                                    */
/* -------------------------------------------------------------------------- */

export const brand = {
  /** MEASURED - nut "+ Create new" cua Admin Movie, avatar, nut "Proceed Payment",
   *  ghe dang chon cua Select Seat. Mot mau duy nhat cho ca admin lan khach. */
  base: '#1DE782',
  /** DERIVED - #1DE782 pha 20% ve trang, dung cho hover. */
  hover: '#4AEC9B',
  /** DERIVED - #1DE782 pha 15% ve den, dung cho :active va vien nhan. */
  active: '#19C46E',
  /** MEASURED - nen muc menu dang chon trong sider cua Admin Movie. */
  soft: '#BDF0C1',
  /** DERIVED - ban nhat hon `soft`, cho nen cua tag va vung duoc chon nhe. */
  softer: '#E3FCEF',
} as const;

/**
 * Chu nam TREN nen brand.base.
 *
 * SAI LECH CO Y so voi Figma: thiet ke de chu TRANG tren #1DE782, ty le tuong
 * phan chi ~1.4:1 - duoi nguong doc duoc cua WCAG rat xa. Doi sang mau toi cho
 * ty le ~9:1. Muon giong het thiet ke thi doi mot dong nay thanh '#FFFFFF'.
 */
export const textOnBrand = '#052E1B';

/* -------------------------------------------------------------------------- */
/* 2. Nen toi cua khu khach hang                                               */
/* -------------------------------------------------------------------------- */

/**
 * Khu khach hang la mot nen den nga xanh phu anh sang xanh la toa tu mot goc.
 * Ba diem dung nay MEASURED tu Home, Select Seat va Order detail - dung chung
 * mot dai mau tren ca ba frame.
 */
export const cinemaBackdrop = {
  /** Diem toi nhat, gan nhu den. */
  base: '#020700',
  /** Diem giua cua vung sang. */
  mid: '#083D1F',
  /** Diem sang nhat cua vung sang, va la nen thanh cong cu duoi Select Seat. */
  glow: '#0C582F',
} as const;

/* -------------------------------------------------------------------------- */
/* 3. Mau ngu nghia                                                            */
/* -------------------------------------------------------------------------- */

export const semantic = {
  /** MEASURED - nut xoa trong bang cua Admin Movie.
   *  Thiet ke con mot do thu hai (#DC0000 o nut Logout phia khach); hai mau qua
   *  gan nhau de tao thanh hai y nghia khac nhau, nen gop ve mot. */
  danger: '#D22F27',
  /** DERIVED - thiet ke khong co trang thai canh bao. Lay ho phach du tuong phan
   *  tren ca nen sang lan nen toi. */
  warning: '#D97706',
  /** DERIVED - dung chinh brand.base: trong ngu canh nay "thanh cong" va "hanh
   *  dong chinh" la cung mot mau, va thiet ke khong tach hai thu do. */
  success: brand.base,
  /** DERIVED - xanh duong trung tinh cho thong bao khong mang tin xau. */
  info: '#2563EB',
} as const;

/* -------------------------------------------------------------------------- */
/* 4. Be mat va vien                                                           */
/* -------------------------------------------------------------------------- */

export const surface = {
  light: {
    /** MEASURED - nen cua moi frame admin. */
    base: '#FFFFFF',
    /** DERIVED - lech mot chut khoi trang de the noi dung con thay duoc canh;
     *  Figma de trang tron va dua vao vien dam hon. */
    sunken: '#FAFAFA',
    /** MEASURED - vien bang trong Admin Movie. */
    border: '#BFBFBF',
    /** DERIVED - vien nhat hon cho the va o nhap, vi #BFBFBF qua dam o day. */
    borderSubtle: '#EBEBEB',
    /** MEASURED - icon va vien cua nut phu (xem/sua) trong Admin Movie. */
    iconMuted: '#767676',
  },
  dark: {
    /** DERIVED - nen cua antd darkAlgorithm. Khu admin o che do toi KHONG dung
     *  nen nga xanh cua khu khach; hai khu la hai khong gian khac nhau. */
    base: '#141414',
    sunken: '#0F0F0F',
    raised: '#1F1F1F',
    border: '#303030',
    borderSubtle: '#262626',
  },
} as const;

/* -------------------------------------------------------------------------- */
/* 5. Ghe ngoi                                                                 */
/* -------------------------------------------------------------------------- */

/**
 * Figma CHI dinh nghia hai trang thai: con trong (trang) va dang chon (xanh).
 * Backend co nhieu hon the - available / held / sold, cong them ghe `is_gap` la
 * cho trong khong phai ghe. Nhung trang thai con lai la DERIVED:
 *
 * - `sold`: xam toi, ro rang la khong bam duoc.
 * - `held`: nguoi khac dang giu, con co the nha ra - ho phach de phan biet voi
 *   sold, vi hai cai nay khac nhau ve hanh vi chu khong chi ve ve ngoai.
 * - `gap`: khong ve gi ca, chi chiem cho trong luoi.
 */
export const seat = {
  /** MEASURED - ghe con trong trong Select Seat. */
  available: '#FFFFFF',
  availableText: '#020700',
  /** MEASURED - ghe C8/C9/C10 dang duoc chon. */
  selected: brand.base,
  selectedText: textOnBrand,
  /** DERIVED */
  sold: '#3A3F3C',
  soldText: '#8A8F8B',
  /** DERIVED */
  held: '#D97706',
  heldText: '#1A1200',
  /** DERIVED - o trong luoi khong phai ghe. */
  gap: 'transparent',
  /** MEASURED - thanh "X" dai dien man chieu trong Select Seat. */
  screen: '#FFFFFF',
} as const;

/* -------------------------------------------------------------------------- */
/* 6. Loai ghe - dung trong trinh sua so do cua admin                          */
/* -------------------------------------------------------------------------- */

/**
 * Backend co DUNG 4 loai ghe (models.AllSeatTypes: standard, vip, couple,
 * recliner) va KHONG co endpoint nao tra ve danh sach do, nen FE phai tu giu.
 *
 * Figma khong ve man sua so do ghe, cung khong phan biet loai ghe o dau ca -
 * man Select Seat chi co "con trong" va "dang chon". Toan bo nhom nay vi vay la
 * DERIVED, tru nen cua `vip` dung lai brand.soft (MEASURED) de khong de ra mot
 * mau xanh thu hai.
 *
 * Day la nen SANG (khu admin), khac han nhom `seat` o tren - nhom do ve tren
 * nen toi cua khu khach va noi ve TINH TRANG ghe (trong/giu/da ban), con nhom
 * nay noi ve LOAI ghe. Hai truc khac nhau, dung tron.
 */
export const seatType = {
  /** DERIVED - xam trung tinh, la loai mac dinh nen phai lang nhat. Chu dung lai
   *  `seat.sold` chu khong phai surface.light.iconMuted: #767676 tren nen nay chi
   *  duoc 3.96:1, duoi nguong 4.5:1 ma test chan. */
  standard: { bg: '#EDF0EE', fg: seat.sold },
  /** bg MEASURED (brand.soft), fg DERIVED - vip la hang cao nen deo mau brand. */
  vip: { bg: brand.soft, fg: '#0B3D22' },
  /** DERIVED - tint cua semantic.warning, du tuong phan voi chu nau dam. */
  couple: { bg: '#FDE8CE', fg: '#7A4405' },
  /** DERIVED - tint cua semantic.info. */
  recliner: { bg: '#DDE7FE', fg: '#1E3A8A' },
} as const;

/* -------------------------------------------------------------------------- */
/* 7. Gom lai                                                                  */
/* -------------------------------------------------------------------------- */

export const tokens = {
  brand,
  textOnBrand,
  cinemaBackdrop,
  semantic,
  surface,
  seat,
  seatType,
} as const;

/**
 * Nen chuyen sac cua khu khach hang. Dung cho <CustomerLayout>; khong dung cho
 * khu admin, vi Figma de khu admin nen trang tron.
 */
export const cinemaGradient =
  `radial-gradient(120% 90% at 8% 40%, ${cinemaBackdrop.glow} 0%, ` +
  `${cinemaBackdrop.mid} 35%, ${cinemaBackdrop.base} 75%)`;
