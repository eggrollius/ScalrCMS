import { useState, useRef, useEffect } from 'react'
import './App.css'

function App() {
  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState('');
  const [visibility, setVisibility] = useState('public');
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState('');
  const [dragActive, setDragActive] = useState(false);
  const [initLoading, setInitLoading] = useState(true);
  const [initError, setInitError] = useState('');
  const [uploadUrl, setUploadUrl] = useState<string | null>(null);
  const [videoId, setVideoId] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const initialize = async () => {
      setInitLoading(true);
      setInitError('');
      try {
        const res = await fetch('http://127.0.0.1:8080/api/videos/initialize', { method: 'POST' });
        if (!res.ok) {
          setInitError('Failed to initialize upload.');
        } else {
          const data = await res.json();
          if (data.success) {
            setUploadUrl(data.uploadUrl);
            setVideoId(data.videoId);
          } else {
            setInitError(data.error || 'Failed to initialize upload.');
          }
        }
      } catch (err) {
        setInitError('Error initializing upload.');
      } finally {
        setInitLoading(false);
      }
    };
    initialize();
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const selectedFile = e.target.files[0];
      setFile(selectedFile);
      uploadToPresignedUrl(selectedFile);
    }
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const droppedFile = e.dataTransfer.files[0];
      setFile(droppedFile);
      uploadToPresignedUrl(droppedFile);
    }
  };

  const handleBrowseClick = () => {
    fileInputRef.current?.click();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) {
      setMessage('Please select a video file.');
      return;
    }
    setUploading(true);
    setMessage('');
    const formData = new FormData();
    formData.append('video', file);
    formData.append('title', title);
    formData.append('description', description);
    formData.append('tags', tags);
    formData.append('visibility', visibility);
    try {
      const res = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });
      if (res.ok) {
        setMessage('Upload successful!');
        setTitle('');
        setDescription('');
        setTags('');
        setVisibility('public');
        setFile(null);
      } else {
        setMessage('Upload failed.');
      }
    } catch (err) {
      setMessage('Error uploading file.');
    } finally {
      setUploading(false);
    }
  };

  const uploadToPresignedUrl = async (file: File) => {
    if (!uploadUrl) {
      setMessage('Upload URL not available.');
      return;
    }
    setUploading(true);
    setMessage('Uploading to S3...');
    try {
      const res = await fetch(uploadUrl, {
        method: 'PUT',
        body: file,
        headers: {
          'Content-Type': file.type || 'application/octet-stream',
        },
      });
      if (res.ok) {
        setMessage('File uploaded to S3!');
      } else {
        setMessage('Failed to upload to S3.');
      }
    } catch (err) {
      setMessage('Error uploading to S3.');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="upload-form-bg">
      {initLoading ? (
        <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh', width: '100%'}}>
          <div className="spinner" aria-label="Loading" />
        </div>
      ) : initError ? (
        <div style={{color: '#dc2626', textAlign: 'center', marginTop: '2rem'}}>{initError}</div>
      ) : (
        <form className="upload-form" onSubmit={handleSubmit}>
          <h1>Upload Video</h1>
          <div className="form-desc">This information will be displayed publicly so be careful what you share.</div>
          <label>
            Title
            <input
              type="text"
              value={title}
              onChange={e => setTitle(e.target.value)}
              required
              maxLength={100}
              placeholder="Enter video title"
            />
          </label>
          <label>
            Description
            <textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              maxLength={1000}
              placeholder="Describe your video"
            />
            <span className="input-hint">Write a few sentences about your video.</span>
          </label>
          <label>
            Tags (comma separated)
            <input
              type="text"
              value={tags}
              onChange={e => setTags(e.target.value)}
              placeholder="e.g. tutorial,react,go"
            />
          </label>
          <label>
            Visibility
            <select value={visibility} onChange={e => setVisibility(e.target.value)}>
              <option value="public">Public</option>
              <option value="unlisted">Unlisted</option>
              <option value="private">Private</option>
            </select>
          </label>
          <div style={{marginBottom: '0.5rem'}}>
            <div style={{fontWeight: 500, marginBottom: 4}}>Video File</div>
            <div
              className={`upload-drop-area${dragActive ? ' dragover' : ''}`}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={handleBrowseClick}
              tabIndex={0}
              role="button"
              style={{outline: 'none'}}
            >
              <div className="upload-icon">📁</div>
              <div className="upload-text">
                {file ? file.name : <span style={{color: '#6366f1', cursor: 'pointer'}}>Upload a file</span>}
              </div>
              <div className="upload-hint">or drag and drop<br/>MP4, MOV, AVI up to 2GB</div>
              <input
                type="file"
                accept="video/*"
                ref={fileInputRef}
                style={{display: 'none'}}
                onChange={handleFileChange}
              />
            </div>
          </div>
          <button type="submit" disabled={uploading}>
            {uploading ? 'Uploading...' : 'Upload Video'}
          </button>
          {message && <p className="upload-message">{message}</p>}
        </form>
      )}
    </div>
  );
}

export default App
