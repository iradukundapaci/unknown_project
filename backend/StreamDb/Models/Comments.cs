using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class Comments : BaseEntity
{
    [Column("message")]
    [Required]
    public  string Message { get; set; }
    
    [Column("user_id")]
    [Required]
    public int UserId { get; set; }
    
    [Column("stream_id")]
    [Required]
    public int StreamId { get; set; }
    
    public User User { get; set; }
    public Streams Stream { get; set; }
}