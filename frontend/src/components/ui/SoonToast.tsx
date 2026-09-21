interface SoonToastProps {
  message: string | null;
}

/** Renderer for useSoonToast. */
export const SoonToast = ({ message }: SoonToastProps) => {
  if (!message) return null;

  return (
    <div
      role="status"
      className="fixed inset-x-0 bottom-4 z-50 mx-auto w-fit rounded-full bg-black/85 px-4 py-2 text-xs text-white shadow-lg"
    >
      {message}
    </div>
  );
};

export default SoonToast;
