"use client"

import { useEffect, useState } from "react"

interface TermsOfServiceProps {
  isOpen: boolean
  onAccept: () => void
}

export function TermsOfService({ isOpen, onAccept }: TermsOfServiceProps) {
  const [hasScrolledToBottom, setHasScrolledToBottom] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setHasScrolledToBottom(false)
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }

    return () => {
      document.body.style.overflow = 'unset'
    }
  }, [isOpen])

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLDivElement
    const bottom = target.scrollHeight - target.scrollTop <= target.clientHeight + 10
    if (bottom && !hasScrolledToBottom) {
      setHasScrolledToBottom(true)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[10000] flex items-center justify-center p-4 bg-black/50">
      <div className="relative w-full max-w-3xl bg-white rounded-2xl shadow-2xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="px-6 py-5 border-b border-gray-200">
          <h2 className="text-2xl font-bold text-gray-900">
            เงื่อนไขการให้บริการ (Terms of Service)
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            โปรดอ่านและยอมรับเงื่อนไขการให้บริการก่อนเริ่มใช้งาน
          </p>
        </div>

        {/* Content */}
        <div 
          className="flex-1 overflow-y-auto px-6 py-4 text-gray-700"
          onScroll={handleScroll}
        >
          <div className="prose prose-sm max-w-none space-y-4">
            <p className="font-medium text-gray-900">
              AILaw: Legal Assistant Chatbot
            </p>
            
            <p className="text-red-600 font-medium">
              โปรดอ่านเงื่อนไขการให้บริการฉบับนี้โดยละเอียดก่อนเริ่มใช้งาน
              การกดปุ่ม "ยอมรับ" หรือการเริ่มใช้งานระบบ ถือว่าผู้ใช้งานตกลงและยอมรับเงื่อนไขทั้งหมดนี้แล้ว
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">1. บทนำและวัตถุประสงค์ของระบบ</h3>
            <p>
              AILaw: Legal Assistant Chatbot ("ระบบ" หรือ "แชทบอท") เป็นโครงงานที่จัดทำขึ้นโดยนักศึกษาในระดับอุดมศึกษา 
              มีวัตถุประสงค์เพื่อพัฒนาเครื่องมือช่วยสืบค้นข้อมูลจากประมวลกฎหมายแพ่งและพาณิชย์ของประเทศไทย 
              โดยประยุกต์ใช้เทคโนโลยีปัญญาประดิษฐ์และ Large Language Model (LLM)
            </p>
            <p className="font-medium">
              ระบบถูกพัฒนาขึ้นเพื่อการศึกษา การวิจัย และการใช้งานเชิงข้อมูลเบื้องต้นเท่านั้น 
              มิได้มีวัตถุประสงค์เพื่อให้คำปรึกษาทางกฎหมายอย่างเป็นทางการ
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">2. คำจำกัดความ</h3>
            <ul className="list-disc pl-6 space-y-2">
              <li>"ผู้ให้บริการ" หมายถึง คณะผู้จัดทำโครงงาน AILaw</li>
              <li>"ผู้ใช้งาน" หมายถึง บุคคลทั่วไปที่เข้าใช้งานระบบ</li>
              <li>"ข้อมูลทางกฎหมาย" หมายถึง ข้อความจากประมวลกฎหมายแพ่งและพาณิชย์ รวมถึงข้อมูลตัวอย่างที่ใช้ในการพัฒนาและทดสอบระบบ</li>
              <li>"LLM" หมายถึง โมเดลปัญญาประดิษฐ์ที่ใช้ในการประมวลผลและสร้างคำตอบ</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">3. ขอบเขตการให้บริการ</h3>
            <ul className="list-disc pl-6 space-y-2">
              <li>ระบบให้บริการสืบค้นข้อมูลจากประมวลกฎหมายแพ่งและพาณิชย์ของประเทศไทย จำนวน 1,755 มาตรา</li>
              <li>ระบบสามารถแสดงข้อความของมาตรากฎหมายและอ้างอิงมาตราที่เกี่ยวข้องกับคำถามของผู้ใช้งาน</li>
              <li>ระบบรองรับการใช้งานเฉพาะภาษาไทย</li>
              <li>การประมวลผลของระบบจะพิจารณาเฉพาะคำถามปัจจุบัน โดยไม่มีการอ้างอิงคำถามหรือคำตอบก่อนหน้า</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">4. ข้อจำกัดของระบบและความถูกต้องของข้อมูล</h3>
            <ul className="list-disc pl-6 space-y-2">
              <li>ระบบใช้เทคโนโลยีปัญญาประดิษฐ์ซึ่งอาจเกิดความคลาดเคลื่อนในการตีความหรือการแสดงผลข้อมูล</li>
              <li>ผู้ให้บริการไม่รับประกันว่าข้อมูลที่แสดงจะถูกต้อง ครบถ้วน หรือเป็นปัจจุบันเสมอ</li>
              <li>ข้อมูลที่ระบบแสดงอาจไม่ครอบคลุมกรณีเฉพาะ หรือข้อยกเว้นทางกฎหมายในสถานการณ์จริง</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">5. ข้อจำกัดความรับผิดทางกฎหมาย (Legal Disclaimer)</h3>
            <p className="font-medium text-red-600">ระบบนี้ไม่ใช่ทนายความ ที่ปรึกษากฎหมาย หรือผู้เชี่ยวชาญทางกฎหมาย</p>
            <p>ข้อมูลที่ได้จากระบบไม่ถือเป็นคำปรึกษาทางกฎหมาย และไม่สามารถใช้แทนความเห็นของผู้เชี่ยวชาญด้านกฎหมายได้</p>
            <p className="font-medium">ผู้ใช้งานไม่ควรนำข้อมูลจากระบบไปใช้:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>ตัดสินใจทางคดี</li>
              <li>ใช้เป็นหลักฐานทางกฎหมาย</li>
              <li>ใช้ในการดำเนินคดีหรือธุรกรรมทางกฎหมายใด ๆ</li>
            </ul>
            <p className="font-medium text-red-600">
              ผู้ให้บริการไม่รับผิดชอบต่อความเสียหายใด ๆ ที่เกิดจากการนำข้อมูลจากระบบไปใช้ ไม่ว่าทางตรงหรือทางอ้อม
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">6. ความเหมาะสมของการใช้งาน</h3>
            <p>ผู้ใช้งานตกลงว่าจะใช้ระบบ:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>เพื่อการศึกษา การค้นคว้า หรือการทำความเข้าใจข้อมูลทางกฎหมายเบื้องต้น</li>
              <li>โดยใช้วิจารณญาณของตนเองและตรวจสอบข้อมูลจากแหล่งทางการเพิ่มเติม</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">7. การใช้งานที่ต้องห้าม</h3>
            <p>ผู้ใช้งานตกลงว่าจะไม่:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>ใช้ระบบเพื่อกระทำการที่ขัดต่อกฎหมาย ศีลธรรม หรือความสงบเรียบร้อย</li>
              <li>ป้อนข้อมูลที่เป็นเท็จ ละเมิดสิทธิส่วนบุคคล หรือสิทธิของผู้อื่น</li>
              <li>พยายามรบกวน เจาะระบบ หรือดัดแปลงการทำงานของระบบ</li>
              <li>ใช้ระบบในเชิงพาณิชย์โดยไม่ได้รับอนุญาต</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">8. ข้อมูลส่วนบุคคลและความเป็นส่วนตัว</h3>
            <p>ระบบมีวัตถุประสงค์เพื่อการศึกษาและการพัฒนาเทคโนโลยี</p>
            <p className="font-medium">ผู้ใช้งานไม่ควรกรอกข้อมูลส่วนบุคคลที่อ่อนไหว เช่น:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>เลขประจำตัวประชาชน</li>
              <li>ข้อมูลทางการเงิน</li>
              <li>ข้อมูลคดีความจริงของตนเองหรือผู้อื่น</li>
            </ul>
            <p>ข้อมูลการใช้งานอาจถูกนำไปใช้เพื่อ:</p>
            <ul className="list-disc pl-6 space-y-2">
              <li>การประเมินประสิทธิภาพของระบบ</li>
              <li>การปรับปรุงและพัฒนาระบบในเชิงวิชาการ</li>
            </ul>

            <h3 className="text-lg font-bold text-gray-900 mt-6">9. ทรัพย์สินทางปัญญา</h3>
            <p>
              ซอฟต์แวร์ ระบบ กระบวนการทำงาน และองค์ประกอบทั้งหมดของ AILaw เป็นทรัพย์สินทางปัญญาของผู้จัดทำโครงงาน
            </p>
            <p>
              ห้ามคัดลอก ดัดแปลง หรือเผยแพร่เพื่อวัตถุประสงค์อื่นโดยไม่ได้รับอนุญาต
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">10. การระงับหรือยุติการให้บริการ</h3>
            <p>
              ผู้ให้บริการขอสงวนสิทธิ์ในการระงับหรือยุติการให้บริการระบบทั้งหมดหรือบางส่วนได้ตลอดเวลา 
              โดยไม่จำเป็นต้องแจ้งให้ทราบล่วงหน้า
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">11. การเปลี่ยนแปลงเงื่อนไข</h3>
            <p>
              ผู้ให้บริการอาจแก้ไขหรือปรับปรุงเงื่อนไขการให้บริการฉบับนี้ได้ตามความเหมาะสม
            </p>
            <p>
              การใช้งานระบบต่อไปหลังจากมีการเปลี่ยนแปลง ถือว่าผู้ใช้งานยอมรับเงื่อนไขที่แก้ไขแล้ว
            </p>

            <h3 className="text-lg font-bold text-gray-900 mt-6">12. กฎหมายที่ใช้บังคับ</h3>
            <p>
              เงื่อนไขการให้บริการฉบับนี้อยู่ภายใต้บังคับแห่งกฎหมายของราชอาณาจักรไทย
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 bg-gray-50">
          <button
            onClick={onAccept}
            disabled={!hasScrolledToBottom}
            className={`w-full px-6 py-3 rounded-xl font-semibold transition-all ${
              hasScrolledToBottom
                ? "bg-black text-white hover:bg-gray-800"
                : "bg-gray-300 text-gray-500 cursor-not-allowed"
            }`}
          >
            {hasScrolledToBottom ? "ยอมรับเงื่อนไขการให้บริการ" : "กรุณาเลื่อนอ่านเงื่อนไขทั้งหมด"}
          </button>
        </div>
      </div>
    </div>
  )
}