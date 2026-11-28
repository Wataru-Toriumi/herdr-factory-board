import AVFoundation
import Foundation
import Speech

// 標準出力の先頭行で録音開始を通知し、その後に確定した本文だけを返す。
func output(_ text: String) {
    FileHandle.standardOutput.write(Data(text.utf8))
}

func fail(_ message: String) -> Never {
    FileHandle.standardError.write(Data(message.utf8))
    exit(1)
}

final class Dictation {
    let engine = AVAudioEngine()
    let request = SFSpeechAudioBufferRecognitionRequest()
    let recognizer = SFSpeechRecognizer(locale: Locale(identifier: "ja-JP"))
    var task: SFSpeechRecognitionTask?
    var recording = false

    func authorize() {
        SFSpeechRecognizer.requestAuthorization { status in
            guard status == .authorized else {
                fail("システム設定の「プライバシーとセキュリティ」で音声認識を許可してください")
            }
            AVCaptureDevice.requestAccess(for: .audio) { granted in
                guard granted else {
                    fail("システム設定の「プライバシーとセキュリティ」でマイクを許可してください")
                }
                DispatchQueue.main.async { self.start() }
            }
        }
    }

    func start() {
        guard let recognizer = recognizer, recognizer.supportsOnDeviceRecognition,
              recognizer.isAvailable else {
            fail("日本語のオンデバイス音声認識を利用できません。Macの音声入力の言語と設定を確認してください")
        }
        recognizer.queue = OperationQueue.main
        request.requiresOnDeviceRecognition = true
        request.shouldReportPartialResults = false
        let input = engine.inputNode
        let format = input.outputFormat(forBus: 0)
        guard format.channelCount > 0, format.sampleRate > 0 else {
            fail("入力用マイクが見つかりません。Macのサウンド設定を確認してください")
        }
        do {
            try installMicrophoneTap(input, format)
            try engine.start()
        } catch {
            fail("録音を開始できません: " + error.localizedDescription)
        }
        task = recognizer.recognitionTask(with: request) { result, error in
            if let result = result, result.isFinal {
                output(result.bestTranscription.formattedString)
                exit(0)
            }
            if let error = error {
                fail("音声を認識できません: " + error.localizedDescription)
            }
        }
        recording = true
        output("ready\n")
    }

    private func installMicrophoneTap(_ input: AVAudioInputNode, _ format: AVAudioFormat) throws {
#if compiler(>=6.4)
        if #available(macOS 27, *) {
            try input.installAudioTap(onBus: 0, bufferSize: 1024, format: format) { buffer, _ in
                self.request.append(AVAudioPCMBuffer(copying: buffer))
            }
        } else {
            installLegacyTap(input, format)
        }
#else
        installLegacyTap(input, format)
#endif
    }

    @available(macOS, deprecated: 27)
    private func installLegacyTap(_ input: AVAudioInputNode, _ format: AVAudioFormat) {
        input.installTap(onBus: 0, bufferSize: 1024, format: format) { buffer, _ in
            self.request.append(buffer)
        }
    }

    func stop() {
        guard recording else { return }
        recording = false
        engine.stop()
        engine.inputNode.removeTap(onBus: 0)
        request.endAudio()
        DispatchQueue.main.asyncAfter(deadline: .now() + 15) {
            fail("文字起こしがタイムアウトしました")
        }
    }
}

let dictation = Dictation()
// ボードが終了してパイプが閉じた場合も、マイクを解放する。
DispatchQueue.global().async {
    while let line = readLine() {
        if line == "stop" {
            DispatchQueue.main.async { dictation.stop() }
        }
    }
    exit(0)
}
dictation.authorize()
RunLoop.main.run()
