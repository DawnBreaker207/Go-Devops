import { useEffect, useRef } from 'react';
import Quill from 'quill';
import 'quill/dist/quill.snow.css';

interface RichTextEditorProps {
  value?: string;
  onChange?: (html: string) => void;
  placeholder?: string;
  minHeight?: number;
}

export const RichTextEditor = ({
  value,
  onChange,
  placeholder,
  minHeight = 200,
}: RichTextEditorProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const initialValueRef = useRef(value);
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const editorEl = document.createElement('div');
    container.appendChild(editorEl);

    const quill = new Quill(editorEl, {
      theme: 'snow',
      placeholder,
      modules: {
        toolbar: [
          [{ header: [2, 3, false] }],
          ['bold', 'italic', 'underline'],
          [{ list: 'ordered' }, { list: 'bullet' }],
          ['link'],
          ['clean'],
        ],
      },
    });
    quill.root.style.minHeight = `${minHeight}px`;

    if (initialValueRef.current) quill.clipboard.dangerouslyPasteHTML(initialValueRef.current);

    quill.on('text-change', () => {
      const html = quill.root.innerHTML;
      onChangeRef.current?.(html === '<p><br></p>' ? '' : html);
    });

    return () => {
      container.replaceChildren();
    };
  }, [placeholder, minHeight]);

  return <div ref={containerRef} />;
};

export default RichTextEditor;
