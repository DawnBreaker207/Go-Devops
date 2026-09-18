import { useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { useMovieDetail, useMovieShowtimes } from './hooks/useBrowse';
import { selectSeatPath } from '@/routes/paths';
import { errorMessage } from '@/utils/error';
import {
  API_DATE_FORMAT,
  CINEMA_TZ,
  formatDuration,
  formatVND,
  toCinemaTime,
} from '@/utils/format';
import './FilmPage.css';

/**
 * Chi tiet phim + o chon suat chieu.
 *
 * `GET /movies/:id/showtimes` KEP ket qua vao DUNG MOT NGAY (mac dinh hom nay),
 * nen man nay bat buoc phai co thanh chon ngay - khong co no thi khach tuong
 * rap chi chieu mot ngay duy nhat.
 */

/** So ngay mo ban hien ra tren thanh chon ngay. */
const DAYS_AHEAD = 7;

export const FilmPage = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();

  // Ngay lam viec theo gio RAP, khong theo gio may nguoi xem: mot khach o mui
  // gio khac se thay "hom nay" lech mot ngay so voi rap neu dung dayjs() tran.
  const days = useMemo(
    () =>
      Array.from({ length: DAYS_AHEAD }, (_, i) =>
        dayjs().tz(CINEMA_TZ).startOf('day').add(i, 'day')
      ),
    []
  );
  const [selectedDay, setSelectedDay] = useState(() => days[0].format(API_DATE_FORMAT));

  const movie = useMovieDetail(id);
  const showtimes = useMovieShowtimes(id, selectedDay);

  if (movie.error) {
    return (
      <div className="cp-notice cp-notice--error" role="alert">
        {errorMessage(movie.error, t('common.somethingWrong'))}
      </div>
    );
  }

  const film = movie.data;

  return (
    <>
      <div className="cp-film">
        <div className="cp-film__poster">
          {film?.poster_url ? <img src={film.poster_url} alt="" /> : null}
        </div>

        <div>
          <h1 className="cp-film__title">{film?.title ?? '…'}</h1>

          <div className="cp-film__tags">
            {film?.age_rating ? (
              <span className="cp-film__tag cp-film__tag--rating">{film.age_rating}</span>
            ) : null}
            {film?.genre ? <span className="cp-film__tag">{film.genre}</span> : null}
            {film?.duration ? (
              <span className="cp-film__tag">{formatDuration(film.duration)}</span>
            ) : null}
          </div>

          {film?.description ? <p className="cp-film__desc">{film.description}</p> : null}

          <div className="cp-film__facts">
            {film?.director ? (
              <div>
                <span className="cp-film__fact-label">{t('movie.director')}</span>
                {film.director}
              </div>
            ) : null}
            {film?.cast ? (
              <div>
                <span className="cp-film__fact-label">{t('movie.cast')}</span>
                {film.cast}
              </div>
            ) : null}
          </div>

          <h2 className="cp-title" style={{ fontSize: 20, marginTop: 0 }}>
            {t('customer.pickShowtime')}
          </h2>

          <div className="cp-daybar" role="group" aria-label={t('customer.pickDay')}>
            {days.map((day) => {
              const key = day.format(API_DATE_FORMAT);
              return (
                <button
                  key={key}
                  type="button"
                  aria-pressed={key === selectedDay}
                  className={`cp-day${key === selectedDay ? ' cp-day--active' : ''}`}
                  onClick={() => setSelectedDay(key)}
                >
                  <span className="cp-day__dow">{day.format('ddd')}</span>
                  <span className="cp-day__num">{day.format('DD/MM')}</span>
                </button>
              );
            })}
          </div>

          {showtimes.error ? (
            <div className="cp-notice cp-notice--error" role="alert">
              {errorMessage(showtimes.error, t('common.somethingWrong'))}
            </div>
          ) : null}

          {showtimes.isFetching ? <p className="cp-muted">{t('common.loading')}</p> : null}

          {!showtimes.isFetching && (showtimes.data?.length ?? 0) === 0 && !showtimes.error ? (
            // Mang rong o day co NHIEU nguyen nhan: phim khong con `showing`,
            // suat da dong, suat da qua gio, hoac phong thieu gia. Backend
            // khong phan biet duoc, nen chi noi chung mot cau trung thuc.
            <p className="cp-empty">{t('customer.noShowtimes')}</p>
          ) : null}

          {(showtimes.data?.length ?? 0) > 0 ? (
            <div className="cp-showtimes">
              {showtimes.data?.map((show) => (
                <Link key={show.id} to={selectSeatPath(show.id)} className="cp-showtime">
                  <span className="cp-showtime__time">
                    {toCinemaTime(show.start_at).format('HH:mm')}
                  </span>
                  <span className="cp-showtime__hall">{show.hall_name}</span>
                  {/* from_price la omitempty: vang mat khi bang 0, tuc phong
                      chua cau hinh gia. Khong ve "0 d" ra man hinh. */}
                  {show.from_price ? (
                    <span className="cp-showtime__price">
                      {t('customer.fromPrice', { price: formatVND(show.from_price) })}
                    </span>
                  ) : null}
                </Link>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </>
  );
};

export default FilmPage;
